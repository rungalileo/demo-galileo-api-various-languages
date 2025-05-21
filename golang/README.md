# Galileo Go Demo

This is a Go implementation of the Galileo API client that demonstrates how to interact with the Galileo API.

## Prerequisites

1. Install Go (version 1.16 or later)
2. Get your Galileo API key from the Galileo console:
   - Go to Galileo console home
   - Click your icon (on the bottom left)
   - API Keys
   - Create one

## Running the Demo

1. Navigate to the golang directory:
```bash
cd golang
```

2. Install the dependencies:
```bash
go mod tidy
```

3. Set the required environment variables:
```bash
export GALILEO_API_KEY=your-api-key
export GALILEO_API_URL=https://api.xyz.rungalileo.io
```

4. Run the Observe demo:
```bash
go run demo_observe.go
```

5. Run the Evaluate demo:
```bash
go run demo_evaluate.go
```

## What the Demo Does

The demo performs the following operations:

1. Logs in to the Galileo API using your API key
2. Creates a new project
3. Creates a new run in the project
4. Logs some sample data to the run

## Code Structure

- `demo.go`: Contains the main implementation of the Galileo API client
  - `GalileoClient`: The main client struct that handles API interactions
  - `Login()`: Authenticates with the Galileo API
  - `CreateProject()`: Creates a new project
  - `CreateRun()`: Creates a new run in a project
  - `CustomLog()`: Logs custom data to a run

## Error Handling

The demo includes comprehensive error handling for all API operations. If any operation fails, the program will print an error message and exit with a non-zero status code.

## Environment Variables

The following environment variables are required:

- `GALILEO_API_KEY`: Your Galileo API key
- `GALILEO_API_URL`: The URL of your Galileo API instance (e.g., https://api.xyz.rungalileo.io)

If either of these variables is not set, the program will display an error message and exit. 