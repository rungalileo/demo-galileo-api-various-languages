package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
)

// Node represents a node in the Galileo chain
type Node struct {
	NodeID            string                 `json:"node_id"`
	NodeType          string                 `json:"node_type"`
	NodeName          string                 `json:"node_name"`
	NodeInput         string                 `json:"node_input"`
	NodeOutput        string                 `json:"node_output"`
	ChainRootID       string                 `json:"chain_root_id"`
	ChainID           string                 `json:"chain_id"`
	Step              int                    `json:"step"`
	HasChildren       bool                   `json:"has_children"`
	Inputs            map[string]interface{} `json:"inputs,omitempty"`
	Prompt            string                 `json:"prompt,omitempty"`
	Response          string                 `json:"response,omitempty"`
	CreationTimestamp int64                  `json:"creation_timestamp,omitempty"`
	FinishReason      string                 `json:"finish_reason,omitempty"`
	Latency           int64                  `json:"latency,omitempty"`
	QueryInputTokens  int                    `json:"query_input_tokens,omitempty"`
	QueryOutputTokens int                    `json:"query_output_tokens,omitempty"`
	QueryTotalTokens  int                    `json:"query_total_tokens,omitempty"`
	Params            map[string]interface{} `json:"params,omitempty"`
	Target            string                 `json:"target,omitempty"`
}

// PromptScorersConfiguration represents the configuration for prompt scoring
type PromptScorersConfiguration struct {
	Latency                        bool `json:"latency,omitempty"`
	Cost                           bool `json:"cost,omitempty"`
	PII                            bool `json:"pii,omitempty"`
	InputPII                       bool `json:"input_pii,omitempty"`
	BLEU                           bool `json:"bleu,omitempty"`
	ROUGE                          bool `json:"rouge,omitempty"`
	ProtectStatus                  bool `json:"protect_status,omitempty"`
	ContextRelevance               bool `json:"context_relevance,omitempty"`
	Toxicity                       bool `json:"toxicity,omitempty"`
	InputToxicity                  bool `json:"input_toxicity,omitempty"`
	Tone                           bool `json:"tone,omitempty"`
	InputTone                      bool `json:"input_tone,omitempty"`
	Sexist                         bool `json:"sexist,omitempty"`
	InputSexist                    bool `json:"input_sexist,omitempty"`
	PromptInjection                bool `json:"prompt_injection,omitempty"`
	AdherenceNLI                   bool `json:"adherence_nli,omitempty"`
	ChunkAttributionUtilizationNLI bool `json:"chunk_attribution_utilization_nli,omitempty"`
	CompletenessNLI                bool `json:"completeness_nli,omitempty"`
	Uncertainty                    bool `json:"uncertainty,omitempty"`
	Factuality                     bool `json:"factuality,omitempty"`
	Groundedness                   bool `json:"groundedness,omitempty"`
	PromptPerplexity               bool `json:"prompt_perplexity,omitempty"`
	ChunkAttributionUtilizationGPT bool `json:"chunk_attribution_utilization_gpt,omitempty"`
	CompletenessGPT                bool `json:"completeness_gpt,omitempty"`
}

// CustomLogRequest represents the request for custom logging
type CustomLogRequest struct {
	Rows                    []Node                    `json:"rows"`
	PromptScorersConfig     PromptScorersConfiguration `json:"prompt_scorers_configuration"`
}

// CreateProjectRequest represents the request for creating a project
type CreateProjectRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
	Type     string `json:"type"`
}

// CreateRunRequest represents the request for creating a run
type CreateRunRequest struct {
	Name     string `json:"name"`
	TaskType string `json:"task_type"`
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

// CreateRunResponse represents the response from creating a run
type CreateRunResponse struct {
	Name          string   `json:"name"`
	ProjectID     string   `json:"project_id"`
	CreatedBy     string   `json:"created_by"`
	NumSamples    int      `json:"num_samples"`
	Winner        bool     `json:"winner"`
	DatasetHash   string   `json:"dataset_hash"`
	ID            string   `json:"id"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	TaskType      int      `json:"task_type"`
	LastUpdatedBy string   `json:"last_updated_by"`
	RunTags       []RunTag `json:"run_tags"`
}

// RunTag represents a tag for a run
type RunTag struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	TagType   string `json:"tag_type"`
	ProjectID string `json:"project_id"`
	RunID     string `json:"run_id"`
	CreatedBy string `json:"created_by"`
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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
		return nil, fmt.Errorf("error marshaling login request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error logging in: %s", string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &loginResp, nil
}

// CreateProject creates a new project
func (c *GalileoClient) CreateProject(authToken string) (*CreateProjectResponse, error) {
	url := fmt.Sprintf("%s/projects", c.rootURL)
	
	reqBody, err := json.Marshal(CreateProjectRequest{
		Name:     fmt.Sprintf("test-nachiket-project-new-golang-2"),
		IsPublic: false,
		Type:     "prompt_evaluation",
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling project request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating project: %s", string(body))
	}

	var projectResp CreateProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&projectResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &projectResp, nil
}

// CreateRun creates a new run
func (c *GalileoClient) CreateRun(authToken, projectID, runName string) (*CreateRunResponse, error) {
	url := fmt.Sprintf("%s/projects/%s/runs", c.rootURL, projectID)
	
	reqBody, err := json.Marshal(CreateRunRequest{
		Name:     runName,
		TaskType: "prompt_chain",
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling run request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating run: %s", string(body))
	}

	var runResp CreateRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &runResp, nil
}

// CustomLog logs custom data to Galileo
func (c *GalileoClient) CustomLog(authToken, projectID, runID string) error {
	url := fmt.Sprintf("%s/projects/%s/runs/%s/chains/ingest", c.rootURL, projectID, runID)

	// Use the same UUID for both nodeID and chainRootID
	uuid := uuid.New().String()

	node := Node{
		NodeID:      uuid,
		NodeType:    "llm",
		NodeName:    "LLM",
		NodeInput:   "Tell me a joke about bears!",
		NodeOutput:  "Here is one: Why did the bear go to the doctor? Because it had a grizzly cough!",
		ChainRootID: uuid,
		ChainID:     uuid, // Same as nodeID and chainRootID
		Step:        0,
		HasChildren: false,
		Latency:     0,
	}

	reqBody, err := json.Marshal(CustomLogRequest{
		Rows: []Node{node},
		PromptScorersConfig: PromptScorersConfiguration{
			Factuality:  true,
			Groundedness: true,
		},
	})
	if err != nil {
		return fmt.Errorf("error marshaling custom log request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}

	// Print the response status and body
	fmt.Printf("Response Status: %s\n", resp.Status)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response Body: %s\n", string(body))
	resp.Body = io.NopCloser(bytes.NewBuffer(body)) 
	
	// Reset the body for later use
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error logging data: %s", string(body))
	}

	return nil
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

	// Create project
	fmt.Println("=== CREATING PROJECT ===")
	projectResp, err := client.CreateProject(loginResp.AccessToken)
	if err != nil {
		fmt.Printf("Error creating project: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("PROJECT CREATED: %s\n", projectResp.Name)

	// Create run
	runName := fmt.Sprintf("test-nachiket-run-golang-new")
	fmt.Println("=== CREATING RUN ===")
	runResp, err := client.CreateRun(loginResp.AccessToken, projectResp.ID, runName)
	if err != nil {
		fmt.Printf("Error creating run: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("RUN CREATED: %s\n", runResp.Name)

	// Custom Log
	fmt.Println("=== LOGGING DATA TO GALILEO ===")
	if err := client.CustomLog(loginResp.AccessToken, projectResp.ID, runResp.ID); err != nil {
		fmt.Printf("Error logging data: %v\n", err)
		os.Exit(1)
	}
} 