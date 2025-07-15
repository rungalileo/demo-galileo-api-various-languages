package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const (
	galileoAPIBaseURL = "https://api.galileo.ai"
)

// --- Public Config Structs ---

type LoggerConfig struct {
	ProjectName   string
	LogStreamName string
	APIKey        string
	AuthMethod    string // "api_key" or "bearer_token"
}

type TraceConfig struct {
	Name     string
	Input    string
	Tags     []string
	Metadata map[string]interface{}
}

type SpanConfig struct {
	Name       string
	Input      interface{}
	Output     interface{}
	DurationNs int64
	Metadata   map[string]interface{}
	Tags       []string
	Error      string
	Type       string // "tool", "retriever", "workflow", "agent"
}

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

type ConcludeConfig struct {
	Output     string
	DurationNs int64
	Tags       []string
}

// --- Native Galileo Structs ---

type GalileoSpan struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Input     interface{}            `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type GalileoTrace struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name,omitempty"`
	Input     string                 `json:"input"`
	Output    string                 `json:"output,omitempty"`
	Spans     []*GalileoSpan         `json:"spans"`
	Metadata  map[string]interface{} `json:"user_metadata,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
}

type LogTracesIngestRequest struct {
	LogStreamID string          `json:"log_stream_id"`
	SessionID   string          `json:"session_id,omitempty"`
	Traces      []*GalileoTrace `json:"traces"`
}

// --- Logger Implementation ---

type Logger struct {
	config       LoggerConfig
	httpClient   *http.Client
	projectID    string
	logStreamID  string
	accessToken  string
	sessionID    string
	mu           sync.Mutex
	traceBuffer  []*GalileoTrace
	currentTrace *GalileoTrace
}

func NewLoggerWithConfig(config LoggerConfig) *Logger {
	if config.APIKey == "" {
		log.Fatal("GALILEO_API_KEY must be provided")
	}
	logger := &Logger{
		config:      config,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		traceBuffer: make([]*GalileoTrace, 0),
	}
	ctx := context.Background()
	var err error

	if config.AuthMethod == "bearer_token" {
		logger.accessToken, err = logger.getAccessToken(ctx)
		if err != nil {
			log.Fatalf("Failed to get access token: %v", err)
		}
	}

	logger.projectID, err = logger.getOrCreateProject(ctx, config.ProjectName)
	if err != nil {
		log.Fatalf("Failed to get or create project: %v", err)
	}
	logger.logStreamID, err = logger.getOrCreateLogStream(ctx, config.LogStreamName)
	if err != nil {
		log.Fatalf("Failed to get or create log stream: %v", err)
	}
	return logger
}

func (l *Logger) setAuthHeader(req *http.Request) {
	if l.config.AuthMethod == "bearer_token" {
		req.Header.Set("Authorization", "Bearer "+l.accessToken)
	} else {
		req.Header.Set("Galileo-API-Key", l.config.APIKey)
	}
}

func (l *Logger) StartSession(name string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	url := fmt.Sprintf("%s/projects/%s/sessions", galileoAPIBaseURL, l.projectID)
	body, _ := json.Marshal(map[string]string{
		"name":          name,
		"log_stream_id": l.logStreamID,
	})

	req, _ := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(body))
	l.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("session creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var sessionResp struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&sessionResp)
	l.sessionID = sessionResp.ID
	fmt.Printf("Started session '%s' with ID: %s\n", name, l.sessionID)
	return l.sessionID, nil
}

func (l *Logger) StartTraceWithContext(_ context.Context, config TraceConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	metadata := config.Metadata
	if len(config.Tags) > 0 {
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["tags"] = strings.Join(config.Tags, ",")
	}

	l.currentTrace = &GalileoTrace{
		ID:        uuid.New().String(),
		Name:      config.Name,
		Input:     config.Input,
		Spans:     make([]*GalileoSpan, 0),
		Metadata:  metadata,
		StartTime: time.Now(),
	}
}

func (l *Logger) AddSpan(config SpanConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.currentTrace == nil {
		log.Println("Warning: AddSpan called without an active trace.")
		return
	}
	startTime := time.Now()
	metadata := config.Metadata
	if len(config.Tags) > 0 {
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["tags"] = strings.Join(config.Tags, ",")
	}
	status := "SUCCESS"
	if config.Error != "" {
		status = "ERROR"
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["error"] = config.Error
	}

	spanType := config.Type
	if spanType == "" {
		spanType = "tool"
	}

	span := &GalileoSpan{
		ID:        uuid.New().String(),
		Name:      config.Name,
		Input:     config.Input,
		Output:    config.Output,
		StartTime: startTime,
		EndTime:   startTime.Add(time.Duration(config.DurationNs)),
		Type:      spanType,
		Status:    status,
		Metadata:  metadata,
	}
	l.currentTrace.Spans = append(l.currentTrace.Spans, span)
}

func (l *Logger) AddLlmSpan(config LlmSpanConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentTrace == nil {
		log.Println("Warning: AddLlmSpan called without an active trace.")
		return
	}
	startTime := time.Now()
	metadata := config.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if len(config.Tags) > 0 {
		metadata["tags"] = strings.Join(config.Tags, ",")
	}
	metadata["model"] = config.Model
	metadata["llm.token_count.input"] = config.NumInputTokens
	metadata["llm.token_count.output"] = config.NumOutputTokens
	metadata["llm.token_count.total"] = config.TotalTokens

	span := &GalileoSpan{
		ID:        uuid.New().String(),
		Name:      "llm-span",
		Input:     config.Input,
		Output:    config.Output,
		StartTime: startTime,
		EndTime:   startTime.Add(time.Duration(config.DurationNs)),
		Type:      "llm",
		Status:    "SUCCESS",
		Metadata:  metadata,
	}
	l.currentTrace.Spans = append(l.currentTrace.Spans, span)
}

func (l *Logger) Conclude(config ConcludeConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentTrace == nil {
		log.Println("Warning: Conclude called without an active trace.")
		return
	}
	l.currentTrace.Output = config.Output
	l.currentTrace.EndTime = l.currentTrace.StartTime.Add(time.Duration(config.DurationNs))
	if len(config.Tags) > 0 {
		if l.currentTrace.Metadata == nil {
			l.currentTrace.Metadata = make(map[string]interface{})
		}
		l.currentTrace.Metadata["completion_tags"] = strings.Join(config.Tags, ",")
	}
	l.traceBuffer = append(l.traceBuffer, l.currentTrace)
	l.currentTrace = nil
}

func (l *Logger) FlushWithContext(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.traceBuffer) == 0 {
		return nil
	}
	ingestRequest := LogTracesIngestRequest{
		LogStreamID: l.logStreamID,
		SessionID:   l.sessionID,
		Traces:      l.traceBuffer,
	}
	body, err := json.Marshal(ingestRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal traces: %w", err)
	}

	url := fmt.Sprintf("%s/projects/%s/traces", galileoAPIBaseURL, l.projectID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create flush request: %w", err)
	}
	l.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to flush traces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("flush failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	l.traceBuffer = make([]*GalileoTrace, 0)
	return nil
}

func (l *Logger) Close() {
	l.FlushWithContext(context.Background())
}

// --- Internal Helper Methods for API Interaction ---

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}
type ProjectDBThin struct{ ID, Name string }
type LogStreamResponse struct{ ID, Name string }

func (l *Logger) getAccessToken(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/login/api_key", galileoAPIBaseURL)
	body, _ := json.Marshal(map[string]string{"api_key": l.config.APIKey})
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	var tokenResp TokenResponse
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	return tokenResp.AccessToken, nil
}

func (l *Logger) getOrCreateProject(ctx context.Context, projectName string) (string, error) {
	url := fmt.Sprintf("%s/projects/all", galileoAPIBaseURL)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	l.setAuthHeader(req)
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var projects []ProjectDBThin
		json.NewDecoder(resp.Body).Decode(&projects)
		for _, p := range projects {
			if p.Name == projectName {
				fmt.Printf("Found existing project '%s' with ID: %s\n", projectName, p.ID)
				return p.ID, nil
			}
		}
	}
	fmt.Printf("Project '%s' not found, creating...\n", projectName)
	return l.createProject(ctx)
}

func (l *Logger) createProject(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/projects", galileoAPIBaseURL)
	body, _ := json.Marshal(map[string]string{"name": l.config.ProjectName, "type": "gen_ai"})
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	l.setAuthHeader(req)
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	var createResp ProjectDBThin
	json.NewDecoder(resp.Body).Decode(&createResp)
	return createResp.ID, nil
}

func (l *Logger) getOrCreateLogStream(ctx context.Context, logStreamName string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/log_streams", galileoAPIBaseURL, l.projectID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	l.setAuthHeader(req)
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var logStreams []LogStreamResponse
		json.NewDecoder(resp.Body).Decode(&logStreams)
		for _, ls := range logStreams {
			if ls.Name == logStreamName {
				fmt.Printf("Found existing log stream '%s' with ID: %s\n", logStreamName, ls.ID)
				return ls.ID, nil
			}
		}
	}
	fmt.Printf("Log stream '%s' not found, creating...\n", logStreamName)
	return l.createLogStream(ctx)
}

func (l *Logger) createLogStream(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/log_streams", galileoAPIBaseURL, l.projectID)
	body, _ := json.Marshal(map[string]string{"name": l.config.LogStreamName})
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	l.setAuthHeader(req)
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	var createResp LogStreamResponse
	json.NewDecoder(resp.Body).Decode(&createResp)
	return createResp.ID, nil
}

// --- Main and Example Functions ---

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}
	config := LoggerConfig{
		ProjectName:   getEnv("GALILEO_PROJECT_NAME", "Default Go Project"),
		LogStreamName: getEnv("GALILEO_LOG_STREAM_NAME", "default-go-stream"),
		APIKey:        getEnv("GALILEO_API_KEY", ""),
		AuthMethod:    getEnv("GALILEO_AUTH_METHOD", "api_key"), // "api_key" or "bearer_token"
	}
	galileoLogger := NewLoggerWithConfig(config)
	defer galileoLogger.Close()

	// Start a session for all the examples
	_, err := galileoLogger.StartSession("Go Demo Session")
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	log.Println("=== Example 1: Basic Trace with LLM Span ===")
	basicTraceExample(galileoLogger)
	log.Println("\n=== Example 2: Advanced Trace with Multiple Spans ===")
	advancedTraceExample(galileoLogger)
	log.Println("\n=== Example 3: RAG Workflow ===")
	ragWorkflowExample(galileoLogger)
	log.Println("\n=== Example 4: Tool Usage ===")
	toolUsageExample(galileoLogger)
	log.Println("\n=== Example 5: Error Handling ===")
	errorHandlingExample(galileoLogger)
	log.Println("\n=== Example 6: Batch Processing ===")
	batchProcessingExample(galileoLogger)
	log.Println("\n=== All examples completed successfully ===")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func basicTraceExample(logger *Logger) {
	logger.StartTraceWithContext(context.Background(), TraceConfig{
		Name:  "Basic LLM Call",
		Input: "What is the capital of France?",
		Tags:  []string{"basic", "llm-only"},
	})
	logger.AddLlmSpan(LlmSpanConfig{
		Input:           "What is the capital of France?",
		Output:          "The capital of France is Paris.",
		Model:           "gpt-4o",
		NumInputTokens:  8,
		NumOutputTokens: 7,
		TotalTokens:     15,
		DurationNs:      1500000000,
		Metadata:        map[string]interface{}{"temperature": 0.7},
		Tags:            []string{"llm", "geography"},
	})
	logger.Conclude(ConcludeConfig{
		Output:     "The capital of France is Paris.",
		DurationNs: 1500000000,
		Tags:       []string{"completed", "success"},
	})
	if err := logger.FlushWithContext(context.Background()); err != nil {
		log.Printf("Error flushing basic trace: %v", err)
	} else {
		log.Println("Basic trace flushed successfully")
	}
}

func advancedTraceExample(logger *Logger) {
	logger.StartTraceWithContext(context.Background(), TraceConfig{
		Name:  "Sentiment Analysis Workflow",
		Input: "Analyze user sentiment",
		Tags:  []string{"advanced", "multi-span"},
	})
	logger.AddSpan(SpanConfig{
		Name:       "data_preprocessing",
		Type:       "tool",
		Input:      "Raw user feedback",
		Output:     "Cleaned feedback",
		DurationNs: 500000000,
		Tags:       []string{"preprocessing"},
	})
	logger.AddLlmSpan(LlmSpanConfig{
		Input:      "Analyze sentiment",
		Output:     "Positive",
		Model:      "gpt-4o",
		DurationNs: 2000000000,
		Tags:       []string{"sentiment-analysis"},
	})
	logger.Conclude(ConcludeConfig{
		Output:     `{"sentiment": "positive"}`,
		DurationNs: 2500000000,
		Tags:       []string{"completed"},
	})
	if err := logger.FlushWithContext(context.Background()); err != nil {
		log.Printf("Error flushing advanced trace: %v", err)
	} else {
		log.Println("Advanced trace flushed successfully")
	}
}

func ragWorkflowExample(logger *Logger) {
	logger.StartTraceWithContext(context.Background(), TraceConfig{
		Name:  "RAG for Quantum Computing",
		Input: "Latest in quantum computing?",
		Tags:  []string{"rag"},
	})
	logger.AddSpan(SpanConfig{
		Name:  "document_retrieval",
		Type:  "retriever",
		Input: "quantum computing",
		Output: []map[string]interface{}{
			{"content": "Document about qubit stability"},
			{"content": "Paper on error correction"},
		},
		DurationNs: 800000000,
		Tags:       []string{"retrieval"},
	})
	logger.AddLlmSpan(LlmSpanConfig{
		Input:      "Summarize documents",
		Output:     "Quantum computing is advancing.",
		Model:      "gpt-4o",
		DurationNs: 3000000000,
	})
	logger.Conclude(ConcludeConfig{
		Output:     "Quantum computing is advancing.",
		DurationNs: 3800000000,
	})
	if err := logger.FlushWithContext(context.Background()); err != nil {
		log.Printf("Error flushing RAG trace: %v", err)
	} else {
		log.Println("RAG trace flushed successfully")
	}
}

func toolUsageExample(logger *Logger) {
	logger.StartTraceWithContext(context.Background(), TraceConfig{
		Name:  "Weather Tool Lookup",
		Input: "Weather in New York?",
		Tags:  []string{"tool-usage"},
	})
	logger.AddSpan(SpanConfig{
		Name:       "weather_tool",
		Type:       "tool",
		Input:      `{"location": "New York"}`,
		Output:     "45°F",
		DurationNs: 1200000000,
		Tags:       []string{"weather-api"},
	})
	logger.AddLlmSpan(LlmSpanConfig{
		Input:      "Format weather: 45°F",
		Output:     "It's 45°F in New York.",
		Model:      "gpt-4o",
		DurationNs: 1000000000,
	})
	logger.Conclude(ConcludeConfig{
		Output:     "It's 45°F in New York.",
		DurationNs: 2200000000,
	})
	if err := logger.FlushWithContext(context.Background()); err != nil {
		log.Printf("Error flushing tool usage trace: %v", err)
	} else {
		log.Println("Tool usage trace flushed successfully")
	}
}

func errorHandlingExample(logger *Logger) {
	logger.StartTraceWithContext(context.Background(), TraceConfig{
		Name:  "API Error and Recovery",
		Input: "Process with potential errors",
		Tags:  []string{"error-handling"},
	})
	logger.AddSpan(SpanConfig{
		Name:       "api_call",
		Type:       "tool",
		Input:      `{"request": "fetch_data"}`,
		Error:      "Connection timeout",
		DurationNs: 5000000000,
		Tags:       []string{"timeout"},
	})
	logger.AddSpan(SpanConfig{
		Name:       "fallback_processing",
		Type:       "tool",
		Input:      `{"source": "cache"}`,
		Output:     "Used cached data",
		DurationNs: 1000000000,
		Tags:       []string{"recovery"},
	})
	logger.Conclude(ConcludeConfig{
		Output:     "Processed with fallback data.",
		DurationNs: 6000000000,
	})
	if err := logger.FlushWithContext(context.Background()); err != nil {
		log.Printf("Error flushing error handling trace: %v", err)
	} else {
		log.Println("Error handling trace flushed successfully")
	}
}

func batchProcessingExample(logger *Logger) {
	items := []string{"item1", "item2", "item3"}
	logger.StartTraceWithContext(context.Background(), TraceConfig{
		Name:  "Batch Item Processing",
		Input: fmt.Sprintf("Process batch of %d items", len(items)),
		Tags:  []string{"batch"},
	})
	for _, item := range items {
		logger.AddSpan(SpanConfig{
			Name:       "process_item",
			Type:       "tool",
			Input:      item,
			Output:     fmt.Sprintf("%s processed", item),
			DurationNs: 500000000,
		})
	}
	logger.Conclude(ConcludeConfig{
		Output:     "Batch processed.",
		DurationNs: 1500000000,
	})
	if err := logger.FlushWithContext(context.Background()); err != nil {
		log.Printf("Error flushing batch trace: %v", err)
	} else {
		log.Println("Batch trace flushed successfully")
	}
}
