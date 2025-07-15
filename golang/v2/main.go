package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// LoggerConfig configures the Galileo logger.
type LoggerConfig struct {
	ProjectID   string
	LogStreamID string
	APIKey      string
	BaseURL     string
	HTTPClient  *http.Client
	DryRun      bool // If true, logs to console instead of sending to API
}

// ProjectV2 represents a Galileo project.
type ProjectV2 struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateProjectResponse wraps the API response for creating a project.
type CreateProjectResponse struct {
	Data ProjectV2 `json:"data"`
}

// SearchProjectsResponse wraps the API response for searching projects.
type SearchProjectsResponse struct {
	Data []ProjectV2 `json:"data"`
}

// LogStreamDB represents a Galileo log stream.
type LogStreamDB struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetLogStreamsResponse wraps the API response for getting log streams.
type GetLogStreamsResponse struct {
	Data []LogStreamDB `json:"data"`
}

// TraceConfig configures a new trace.
type TraceConfig struct {
	Input string
	Tags  []string
}

// SpanConfig configures a generic span.
type SpanConfig struct {
	Name       string
	Input      interface{}
	Output     interface{}
	DurationNs int64
	Tags       []string
	Metadata   map[string]interface{}
}

// LlmSpanConfig configures an LLM span.
type LlmSpanConfig struct {
	Input           string
	Output          string
	Model           string
	NumInputTokens  int
	NumOutputTokens int
	TotalTokens     int
	DurationNs      int64
	Metadata        map[string]interface{}
	Tags            []string
}

// ConcludeConfig provides the final details for a trace.
type ConcludeConfig struct {
	Output     string
	DurationNs int64
	Tags       []string
}

// Span represents a single operation within a trace.
type Span struct {
	Name       string                 `json:"name"`
	Input      interface{}            `json:"input"`
	Output     interface{}            `json:"output"`
	DurationNs int64                  `json:"duration_ns"`
	Metadata   map[string]interface{} `json:"metadata"`
	Tags       []string               `json:"tags"`
}

// Trace represents a complete workflow.
type Trace struct {
	Input      string                 `json:"input"`
	Output     string                 `json:"output,omitempty"`
	DurationNs int64                  `json:"duration_ns,omitempty"`
	Spans      []*Span                `json:"spans"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Logger is the main struct for logging traces to Galileo.
type Logger struct {
	config       LoggerConfig
	currentTrace *Trace
	mu           sync.RWMutex
}

// NewLoggerWithConfig creates a new Galileo logger with the given configuration.
func NewLoggerWithConfig(config LoggerConfig) *Logger {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.galileo.ai/v2" // Correct V2 base path
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	// DryRun is true if APIKey is missing
	if config.APIKey == "" {
		log.Println("GALILEO_API_KEY not set. Running in dry run mode.")
		config.DryRun = true
	}
	// Or if ProjectID is missing
	if !config.DryRun && config.ProjectID == "" {
		log.Println("GALILEO_PROJECT_ID is not set, but API key is present. Running in dry run mode.")
		config.DryRun = true
	}
	// Or if LogStreamID is missing
	if !config.DryRun && config.LogStreamID == "" {
		log.Println("GALILEO_LOG_STREAM_ID could not be resolved. Running in dry run mode.")
		config.DryRun = true
	}
	return &Logger{config: config}
}

// StartTraceWithContext starts a new trace.
func (l *Logger) StartTraceWithContext(ctx context.Context, config TraceConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentTrace = &Trace{
		Input: config.Input,
		Tags:  config.Tags,
		Spans: []*Span{},
	}
}

// AddSpan adds a generic span to the current trace.
func (l *Logger) AddSpan(config SpanConfig) *Span {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.currentTrace == nil {
		return nil
	}
	span := &Span{
		Name:       config.Name,
		Input:      config.Input,
		Output:     config.Output,
		DurationNs: config.DurationNs,
		Metadata:   config.Metadata,
		Tags:       config.Tags,
	}
	l.currentTrace.Spans = append(l.currentTrace.Spans, span)
	return span
}

// AddLlmSpan adds an LLM span to the current trace.
func (l *Logger) AddLlmSpan(config LlmSpanConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.currentTrace == nil {
		return
	}
	metadata := config.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["model"] = config.Model
	metadata["num_input_tokens"] = config.NumInputTokens
	metadata["num_output_tokens"] = config.NumOutputTokens
	metadata["total_tokens"] = config.TotalTokens

	span := &Span{
		Name:       "llm",
		Input:      config.Input,
		Output:     config.Output,
		DurationNs: config.DurationNs,
		Metadata:   metadata,
		Tags:       config.Tags,
	}
	l.currentTrace.Spans = append(l.currentTrace.Spans, span)
}

// Conclude finalizes the trace with output and duration.
func (l *Logger) Conclude(config ConcludeConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.currentTrace == nil {
		return
	}
	l.currentTrace.Output = config.Output
	l.currentTrace.DurationNs = config.DurationNs
	l.currentTrace.Tags = append(l.currentTrace.Tags, config.Tags...)
}

// FlushWithContext sends the completed trace to Galileo or logs it.
func (l *Logger) FlushWithContext(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.currentTrace == nil {
		return fmt.Errorf("no trace to flush")
	}

	// Use a copy of the trace for flushing so we can reset the current one.
	traceToFlush := l.currentTrace
	l.currentTrace = nil

	// The API likely expects a list of events, even if it's just one.
	events := []interface{}{traceToFlush}
	traceData, err := json.MarshalIndent(map[string]interface{}{"events": events}, "", "  ")

	if err != nil {
		// Restore the trace if flushing fails before the request is made
		l.currentTrace = traceToFlush
		return fmt.Errorf("failed to serialize trace: %w", err)
	}

	if l.config.DryRun {
		log.Println("Dry Run: Trace data:")
		log.Println(string(traceData))
		return nil
	}

	url := fmt.Sprintf("%s/projects/%s/log_streams/%s/events", l.config.BaseURL, l.config.ProjectID, l.config.LogStreamID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(traceData))
	if err != nil {
		l.currentTrace = traceToFlush // Restore trace
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.config.APIKey)

	resp, err := l.config.HTTPClient.Do(req)
	if err != nil {
		l.currentTrace = traceToFlush // Restore trace
		return fmt.Errorf("failed to send trace to Galileo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		l.currentTrace = traceToFlush // Restore trace
		return fmt.Errorf("failed to log trace, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// makeAPIRequest is a helper to make authenticated requests to the Galileo API.
func makeAPIRequest(ctx context.Context, method, url, apiKey string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

// getProjectIDByName searches for a project by name and returns its ID.
func getProjectIDByName(ctx context.Context, apiKey, projectName string) (string, error) {
	listURL := fmt.Sprintf("https://api.galileo.ai/v2/projects?name=%s", url.QueryEscape(projectName))
	resp, err := makeAPIRequest(ctx, "GET", listURL, apiKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to list projects, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var listResponse SearchProjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		return "", fmt.Errorf("failed to decode project list response: %w", err)
	}

	// The API might return multiple projects if the name filter is not an exact match.
	// Find the exact match.
	for _, project := range listResponse.Data {
		if project.Name == projectName {
			return project.ID, nil
		}
	}

	return "", nil // Not found
}

// createProject creates a new project and returns its ID.
func createProject(ctx context.Context, apiKey, projectName string) (string, error) {
	createBody := map[string]string{
		"name": projectName,
	}
	bodyBytes, _ := json.Marshal(createBody)

	resp, err := makeAPIRequest(ctx, "POST", "https://api.galileo.ai/v2/projects", apiKey, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create project, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var createResponse CreateProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
		return "", fmt.Errorf("failed to decode create project response: %w", err)
	}

	return createResponse.Data.ID, nil
}

// getLogStreamIDByName searches for a log stream by name within a project.
func getLogStreamIDByName(ctx context.Context, apiKey, projectID, logStreamName string) (string, error) {
	url := fmt.Sprintf("https://api.galileo.ai/v2/projects/%s/log_streams?name=%s", projectID, logStreamName)
	resp, err := makeAPIRequest(ctx, "GET", url, apiKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get log streams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get log streams, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var getResponse GetLogStreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&getResponse); err != nil {
		return "", fmt.Errorf("failed to decode get log streams response: %w", err)
	}

	if len(getResponse.Data) > 0 {
		return getResponse.Data[0].ID, nil
	}

	return "", nil // Not found
}

// createLogStream creates a new log stream within a project.
func createLogStream(ctx context.Context, apiKey, projectID, logStreamName string) (string, error) {
	createBody := map[string]string{"name": logStreamName}
	bodyBytes, _ := json.Marshal(createBody)

	url := fmt.Sprintf("https://api.galileo.ai/v2/projects/%s/log_streams", projectID)
	resp, err := makeAPIRequest(ctx, "POST", url, apiKey, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create log stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create log stream, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var createResponse LogStreamDB
	if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
		return "", fmt.Errorf("failed to decode create log stream response: %w", err)
	}

	return createResponse.ID, nil
}

// getOrCreateLogStreamID ensures a log stream exists and returns its ID.
func getOrCreateLogStreamID(ctx context.Context, apiKey, projectID, logStreamName string) (string, error) {
	if projectID == "" {
		return "", fmt.Errorf("project ID is required to get or create a log stream")
	}
	if logStreamName == "" {
		return "", fmt.Errorf("log stream name is required")
	}

	logStreamID, err := getLogStreamIDByName(ctx, apiKey, projectID, logStreamName)
	if err != nil {
		return "", fmt.Errorf("error searching for log stream: %w", err)
	}

	if logStreamID != "" {
		log.Printf("Found existing log stream '%s' with ID: %s", logStreamName, logStreamID)
		return logStreamID, nil
	}

	log.Printf("Log stream '%s' not found, creating a new one...", logStreamName)
	newLogStreamID, err := createLogStream(ctx, apiKey, projectID, logStreamName)
	if err != nil {
		return "", fmt.Errorf("error creating log stream: %w", err)
	}

	log.Printf("Successfully created new log stream '%s' with ID: %s", logStreamName, newLogStreamID)
	return newLogStreamID, nil
}

// getOrCreateProjectID ensures a project exists and returns its ID.
func getOrCreateProjectID(ctx context.Context, apiKey, projectName string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("API key is required to get or create a project")
	}
	if projectName == "" {
		return "", fmt.Errorf("project name is required")
	}

	projectID, err := getProjectIDByName(ctx, apiKey, projectName)
	if err != nil {
		return "", fmt.Errorf("error searching for project: %w", err)
	}

	if projectID != "" {
		log.Printf("Found existing project '%s' with ID: %s", projectName, projectID)
		return projectID, nil
	}

	log.Printf("Project '%s' not found, creating a new one...", projectName)
	newProjectID, err := createProject(ctx, apiKey, projectName)
	if err != nil {
		return "", fmt.Errorf("error creating project: %w", err)
	}

	log.Printf("Successfully created new project '%s' with ID: %s", projectName, newProjectID)
	return newProjectID, nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using system environment variables")
	}

	ctx := context.Background()
	projectID := getEnv("GALILEO_PROJECT_ID", "")
	projectName := getEnv("GALILEO_PROJECT_NAME", "my-go-v2-project") // Default project name
	apiKey := getEnv("GALILEO_API_KEY", "")

	// If no project ID is provided, try to get or create one by name
	if projectID == "" && apiKey != "" {
		var err error
		projectID, err = getOrCreateProjectID(ctx, apiKey, projectName)
		if err != nil {
			log.Fatalf("Failed to get or create project: %v", err)
		}
	}

	logStreamID := getEnv("GALILEO_LOG_STREAM_ID", "")
	logStreamName := getEnv("GALILEO_LOG_STREAM", "production")

	// If no log stream ID is provided, try to get or create one by name
	if logStreamID == "" && apiKey != "" && projectID != "" {
		var err error
		logStreamID, err = getOrCreateLogStreamID(ctx, apiKey, projectID, logStreamName)
		if err != nil {
			log.Fatalf("Failed to get or create log stream: %v", err)
		}
	}

	// Initialize Galileo logger
	config := LoggerConfig{
		ProjectID:   projectID,
		LogStreamID: logStreamID,
		APIKey:      apiKey,
	}

	galileoLogger := NewLoggerWithConfig(config)

	// Run the LLM with Tool Usage example
	llmAndToolExample(galileoLogger)

	log.Println("\n=== Example completed successfully ===")
}

// Example: LLM with Tool Usage
func llmAndToolExample(logger *Logger) {
	log.Println("\n=== Example: LLM with Tool Usage ===")
	ctx := context.Background()
	logger.StartTraceWithContext(ctx, TraceConfig{
		Input: "What is the weather in London?",
		Tags:  []string{"llm-with-tool", "weather"},
	})

	// Add an LLM span that decides to use a tool and formats a tool call
	logger.AddLlmSpan(LlmSpanConfig{
		Input: "What is the weather in London?",
		// The output contains a tool call. Some models can return structured data for tool calls.
		Output:         `{"tool_call": {"name": "get_weather", "arguments": {"location": "London"}}}`,
		Model:          "gpt-4o-tool-calling",
		NumInputTokens: 10,
		NumOutputTokens: 20,
		TotalTokens:    30,
		DurationNs:     800000000, // 0.8 seconds
		Metadata: map[string]interface{}{
			"model":       "gpt-4o-tool-calling",
			"temperature": 0.1,
		},
		Tags: []string{"llm", "tool-call"},
	})

	// Add a tool span for getting the weather
	logger.AddSpan(SpanConfig{
		Name:       "get_weather",
		Input:      `{"location": "London"}`,
		Output:     `{"temperature": "15°C", "conditions": "Cloudy"}`,
		DurationNs: 500000000, // 0.5 seconds
		Tags:       []string{"tool", "weather-api"},
	})

	// Conclude the trace. In a real scenario, there would be another LLM call to synthesize the final answer.
	// For this example, we'll just conclude with the tool's output.
	logger.Conclude(ConcludeConfig{
		Output:     `{"temperature": "15°C", "conditions": "Cloudy"}`,
		DurationNs: 1300000000, // 1.3 seconds total
		Tags:       []string{"completed", "tool-success"},
	})

	// Flush with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := logger.FlushWithContext(ctx); err != nil {
		log.Printf("Error flushing LLM with tool trace: %v", err)
	} else {
		log.Println("LLM with tool trace flushed successfully")
	}
}

// Helper function to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
