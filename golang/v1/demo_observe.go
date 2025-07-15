package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// CreateProjectRequest represents the request for creating a project
type CreateProjectRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
	Type     string `json:"type"`
}

// CreateProjectResponse represents the response from creating a project
type CreateProjectResponse struct {
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
	IsPublic  bool   `json:"is_public"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// LoginRequest represents the request for logging in
type LoginRequest struct {
	APIKey string `json:"api_key"`
}

// LoginResponse represents the response from login
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// CreateAlertRequest represents the request for creating an alert
type CreateAlertRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Conditions  []AlertCondition       `json:"conditions"`
	Interval    int                    `json:"interval"`
	Channels    []AlertChannel         `json:"channels"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Enabled     bool                   `json:"enabled"`
}

// AlertCondition represents the condition for an alert
type AlertCondition struct {
	Field         string      `json:"field"`
	Aggregation   string      `json:"aggregation"`
	Operator      string      `json:"operator"`
	Value         interface{} `json:"value"`
	FilterValue   interface{} `json:"filter_value,omitempty"`
	FilterOperator string      `json:"filter_operator,omitempty"`
	Window        int         `json:"window"`
	ConditionType string      `json:"condition_type,omitempty"`
}

// AlertChannel represents a channel for an alert
type AlertChannel struct {
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

// CreateAlertResponse represents the response from creating an alert
type CreateAlertResponse struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"project_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Conditions  []AlertCondition       `json:"conditions"`
	Interval    int                    `json:"interval"`
	Channels    []AlertChannel         `json:"channels"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	Enabled     bool                   `json:"enabled"`
	CreatedBy   string                 `json:"created_by"`
}

// Document represents a RAG document
type Document struct {
	PageContent string                 `json:"page_content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowStep represents a step in a workflow
type WorkflowStep struct {
	Type         string                 `json:"type"`
	Input        interface{}            `json:"input"`
	Output       interface{}            `json:"output,omitempty"`
	Name         string                 `json:"name,omitempty"`
	CreatedAtNs  int64                  `json:"created_at_ns,omitempty"`
	DurationNs   int64                  `json:"duration_ns,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	StatusCode   interface{}            `json:"status_code,omitempty"`
	GroundTruth  interface{}            `json:"ground_truth,omitempty"`
	Steps        []interface{}          `json:"steps,omitempty"`
	Parent       interface{}            `json:"parent,omitempty"`
}

// WorkflowLogRequest represents the request to log workflows
type WorkflowLogRequest struct {
	Workflows   []WorkflowStep `json:"workflows"`
	ProjectID   string         `json:"project_id,omitempty"`
	ProjectName string         `json:"project_name,omitempty"`
}

// GalileoClient represents the Galileo API client
type GalileoClient struct {
	rootURL string
	apiKey  string
	client  *http.Client
}

// NewGalileoClient creates a new Galileo API client
func NewGalileoClient(rootURL, apiKey string) *GalileoClient {
	return &GalileoClient{
		rootURL: rootURL,
		apiKey:  apiKey,
		client:  &http.Client{},
	}
}

// Login authenticates with the Galileo API
func (c *GalileoClient) Login() (*LoginResponse, error) {
	url := fmt.Sprintf("%s/login/api_key", c.rootURL)
	
	reqBody, err := json.Marshal(LoginRequest{APIKey: c.apiKey})
	if err != nil {
		return nil, fmt.Errorf("error marshaling login request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error logging in: %s", string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &loginResp, nil
}

// CreateMonitorProject creates a new project with type llm_monitor
func (c *GalileoClient) CreateMonitorProject(authToken string) (*CreateProjectResponse, error) {
	url := fmt.Sprintf("%s/projects", c.rootURL)
	
	projectData := CreateProjectRequest{
		Name:     fmt.Sprintf("golang-llm-monitor-project-%d", time.Now().Unix()),
		IsPublic: false,
		Type:     "llm_monitor",
	}
	
	reqBody, err := json.Marshal(projectData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling project request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("error creating monitor project (status %d): %s", resp.StatusCode, string(body))
	}

	var projectResp CreateProjectResponse
	if err := json.Unmarshal(body, &projectResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &projectResp, nil
}

// debugHTTP prints the request and response details for debugging
func debugHTTP(reqBody []byte, resp *http.Response, body []byte) {
	fmt.Printf("Request Body: %s\n", string(reqBody))
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body: %s\n", string(body))
}

// CreateAlert creates a new alert for a project
func (c *GalileoClient) CreateAlert(authToken, projectID string) (*CreateAlertResponse, error) {
	url := fmt.Sprintf("%s/projects/%s/alerts/create", c.rootURL, projectID)
	
	// Email configuration - in a real application, replace with actual email
	emailConfig := map[string]interface{}{
		"recipients": []string{"new-user@galileo.ai"},
	}

	reqBodyData := CreateAlertRequest{
		Name:        "High PII Detection Alert",
		Description: "Alert when PII content is detected in LLM responses",
		Tags:        []string{"security", "pii", "privacy"},
		Conditions: []AlertCondition{
			{
				Field:         "score_pii",        // The field to monitor
				Aggregation:   "avg",              // Aggregate by average value
				Operator:      "gt",               // greater than
				Value:         0.7,                // Alert when PII score is over 0.7
				Window:        900,                // 15 minutes (in seconds)
				ConditionType: "metric/numeric/1", // The type of condition
			},
		},
		Interval: 300,  // Check every 5 minutes (in seconds)
		Channels: []AlertChannel{
			{
				Type:    "email",
				Config:  emailConfig,
				Enabled: true,
			},
		},
		Enabled: true,
	}

	reqBody, err := json.Marshal(reqBodyData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling alert request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	
	// Debug info
	debugHTTP(reqBody, resp, body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("error creating alert (status %d): %s", resp.StatusCode, string(body))
	}

	var alertResp CreateAlertResponse
	if err := json.Unmarshal(body, &alertResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &alertResp, nil
}

// LogWorkflows logs workflows to a Galileo Observe project
func (c *GalileoClient) LogWorkflows(authToken string, request WorkflowLogRequest) error {
	url := fmt.Sprintf("%s/observe/workflows", c.rootURL)
	
	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling workflow log request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}
	
	// Debug info
	debugHTTP(reqBody, resp, body)
	
	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("error logging workflows (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// DemoLogWorkflows demonstrates workflow logging
func (c *GalileoClient) DemoLogWorkflows(authToken string, projectID string) error {
	// Create a timestamp for the workflow
	now := time.Now()
	timestampNs := now.UnixNano()
	startTime := timestampNs - 1000000000 // 1 second ago
	
	// Create a simple workflow with one step
	request := WorkflowLogRequest{
		Workflows: []WorkflowStep{
			{
				Type:  "agent",
				Name:  "Simple LLM Query",
				Input: "What is the capital of France?",
				Output: "The capital of France is Paris. Paris is known for its art, culture, " +
					"cuisine, and historical monuments like the Eiffel Tower.",
				CreatedAtNs: startTime,
				DurationNs:  1000000000, // 1 second in nanoseconds
				Metadata: map[string]interface{}{
					"model":      "gpt-4",
					"user_id":    "test-user-123",
					"session_id": "test-session-456",
					"tags":       "demo,golang,observe", // Joined as a string
				},
				StatusCode: 200,
				Steps: []interface{}{
					map[string]interface{}{
						"type":          "llm",
						"name":          "LLM Call",
						"input":         "What is the capital of France?",
						"output":        "Paris is the capital of France.",
						"created_at_ns": startTime + 100000000,
						"duration_ns":   800000000, // 800ms
						"metadata": map[string]interface{}{
							"model":             "gpt-4",
							"prompt_tokens":     "10",   // String version for numeric values
							"completion_tokens": "8",    
							"total_tokens":      "18",   
						},
					},
				},
			},
		},
		ProjectID: projectID,
	}
	
	return c.LogWorkflows(authToken, request)
}

// DemoLogRAGWorkflows shows an example of logging RAG workflows
func (c *GalileoClient) DemoLogRAGWorkflows(authToken string, projectID string) error {
	// Create timestamps for the workflow
	now := time.Now()
	timestampNs := now.UnixNano()
	startTime := timestampNs - 3000000000 // 3 seconds ago
	
	// Create retriever output in the correct format
	// Manually build the document objects to ensure correct field names
	docs := []map[string]interface{}{
		{
			"page_content": "Paris is the capital and most populous city of France. It has an estimated population of 2,165,423 residents as of 2019 in an area of more than 105 square kilometers.",
			"metadata": map[string]interface{}{
				"source": "geography_database",
				"score":  "0.92",
			},
		},
		{
			"page_content": "Paris is known worldwide for its art museums, fashion scene, and iconic landmarks like the Eiffel Tower, Louvre, and Notre-Dame Cathedral.",
			"metadata": map[string]interface{}{
				"source": "travel_guide",
				"score":  "0.85",
			},
		},
	}
	
	// Create a RAG workflow with multiple steps
	request := WorkflowLogRequest{
		Workflows: []WorkflowStep{
			{
				Type:  "agent",
				Name:  "RAG Query Process",
				Input: "Tell me about Paris, France.",
				Output: "Paris is the capital and most populous city of France, with over 2 million residents. " +
					"It's renowned for its art museums, fashion scene, and iconic landmarks including the Eiffel Tower, " +
					"Louvre Museum, and Notre-Dame Cathedral. The city covers more than 105 square kilometers and " +
					"is considered one of the world's major cultural and historical centers.",
				CreatedAtNs: startTime,
				DurationNs:  3000000000, // 3 seconds in nanoseconds
				Metadata: map[string]interface{}{
					"model":      "gpt-4",
					"user_id":    "test-user-789",
					"session_id": "test-session-456",
					"tags":       "demo,golang,observe,rag", // Joined as a string
					"tracing_id": "trace-abc-123",
				},
				StatusCode: 200,
				Steps: []interface{}{
					map[string]interface{}{
						"type":          "retriever", 
						"name":          "Vector Store Query",
						"input":         "Paris, France",
						"output":        docs,
						"created_at_ns": startTime + 100000000,
						"duration_ns":   1200000000, // 1.2s
						"metadata": map[string]interface{}{
							"vector_store": "pinecone",
							"index_name":   "knowledge_base",
							"top_k":        "2", // String version for numeric values
							"similarity":   "cosine",
						},
					},
					map[string]interface{}{
						"type":          "llm",
						"name":          "Answer Generation with Context",
						"input":         "Question: Tell me about Paris, France.\nContext: Paris is the capital and most populous city of France. It has an estimated population of 2,165,423 residents as of 2019 in an area of more than 105 square kilometers. Paris is known worldwide for its art museums, fashion scene, and iconic landmarks like the Eiffel Tower, Louvre, and Notre-Dame Cathedral.\nInstructions: Use the provided context to answer the question accurately.",
						"output":        "Paris is the capital and most populous city of France, with over 2 million residents. It's renowned for its art museums, fashion scene, and iconic landmarks including the Eiffel Tower, Louvre Museum, and Notre-Dame Cathedral. The city covers more than 105 square kilometers and is considered one of the world's major cultural and historical centers.",
						"created_at_ns": startTime + 1500000000,
						"duration_ns":   1400000000, // 1.4s
						"metadata": map[string]interface{}{
							"model":             "gpt-4",
							"prompt_tokens":     "450", // String version for numeric values
							"completion_tokens": "75",  
							"total_tokens":      "525", 
							"temperature":       "0.2", 
							"max_tokens":        "300", 
						},
					},
				},
			},
		},
		ProjectID: projectID,
	}
	
	return c.LogWorkflows(authToken, request)
}

func main() {
	apiKey := os.Getenv("GALILEO_API_KEY")
	rootURL := os.Getenv("GALILEO_API_URL")

	if apiKey == "" || rootURL == "" {
		fmt.Println("Error: GALILEO_API_KEY and GALILEO_API_URL environment variables must be set")
		fmt.Println("Example:")
		fmt.Println("  export GALILEO_API_KEY=your-api-key")
		fmt.Println("  export GALILEO_API_URL=https://api.xyz.rungalileo.io")
		os.Exit(1)
	}

	client := NewGalileoClient(rootURL, apiKey)

	// Login
	fmt.Println("=== LOGGING IN ===")
	loginResp, err := client.Login()
	if err != nil {
		fmt.Printf("Error logging in: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Login successful, received access token")

	// Create LLM monitor project
	fmt.Println("=== CREATING LLM MONITOR PROJECT ===")
	projectResp, err := client.CreateMonitorProject(loginResp.AccessToken)
	if err != nil {
		fmt.Printf("Error creating LLM monitor project: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("MONITOR PROJECT CREATED: %s (ID: %s)\n", projectResp.Name, projectResp.ID)

	// Create alert
	fmt.Println("=== CREATING ALERT ===")
	alertResp, err := client.CreateAlert(loginResp.AccessToken, projectResp.ID)
	if err != nil {
		fmt.Printf("Error creating alert: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ALERT CREATED: %s (ID: %s)\n", alertResp.Name, alertResp.ID)

	// Log simple workflow
	fmt.Println("=== LOGGING SIMPLE WORKFLOW ===")
	err = client.DemoLogWorkflows(loginResp.AccessToken, projectResp.ID)
	if err != nil {
		fmt.Printf("Error logging simple workflow: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Simple workflow logged successfully")
	
	// Log RAG workflow
	fmt.Println("=== LOGGING RAG WORKFLOW ===")
	err = client.DemoLogRAGWorkflows(loginResp.AccessToken, projectResp.ID)
	if err != nil {
		fmt.Printf("Error logging RAG workflow: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("RAG workflow logged successfully")
} 