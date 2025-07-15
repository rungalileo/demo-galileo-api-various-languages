# Go Example for Galileo API

This repository provides a Go example demonstrating how to log traces to the Galileo platform using a custom logger that wraps the OpenTelemetry SDK. It showcases various logging scenarios, including basic traces, multi-span workflows, RAG, tool usage, error handling, and batch processing.

## Prerequisites

- Go 1.18 or later installed.
- A valid Galileo API key. You can create one from your Galileo dashboard.

## How to Run

1.  **Create a `.env` file:**

    Create a file named `.env` in the `go-example` directory and add your Galileo credentials. You can also set these as environment variables.

    ```env
    # Your Galileo API key
    GALILEO_API_KEY="your-api-key"

    # (Optional) The authentication method to use. Can be "api_key" or "bearer_token".
    # Defaults to "api_key".
    GALILEO_AUTH_METHOD="api_key"

    # (Optional) Project and Log Stream names
    GALILEO_PROJECT_NAME="My Go Test Project"
    GALILEO_LOG_STREAM_NAME="my-go-test-stream"
    ```

2.  **Run the example:**

    Navigate to the `go-example` directory and run the `main.go` file. The program will execute all the example functions and log the corresponding traces to your Galileo project.

    ```bash
    cd go-example
    go mod tidy # To ensure all dependencies are present
    go run main.go
    ```

## What the Example Does

The demo is structured around a `Logger` component that simplifies interaction with Galileo:

-   **Initialization**: The `Logger` is initialized with your project name, log stream name, and API key. It handles authentication and automatically finds or creates the necessary project and log stream in your Galileo account.
-   **Authentication**: The logger supports two authentication methods, configurable via the `GALILEO_AUTH_METHOD` environment variable:
    -   `"api_key"` (default): Uses the API key directly for all requests.
    -   `"bearer_token"`: Exchanges the API key for a short-lived access token, which is then used for subsequent requests.
-   **Abstraction**: It provides high-level methods like `StartTraceWithContext`, `AddLlmSpan`, `AddSpan`, and `Conclude` that abstract away the complexities of the Galileo API.
-   **Example Scenarios**: The `main` function calls several example functions, each demonstrating a different logging pattern:
    -   **`basicTraceExample`**: A simple trace with a single LLM span.
    -   **`advancedTraceExample`**: A more complex trace with multiple, dependent spans.
    -   **`ragWorkflowExample`**: A Retrieval-Augmented Generation (RAG) workflow with retriever and LLM spans.
    -   **`toolUsageExample`**: A trace demonstrating how to log the use of external tools.
    -   **`errorHandlingExample`**: Shows how to record errors and log recovery steps.
    -   **`batchProcessingExample`**: An example of logging multiple items processed in a batch.

After running the example, you should see the corresponding projects, log streams, and traces in your Galileo UI.