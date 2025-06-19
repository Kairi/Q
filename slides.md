
---
marp: true
theme: gaia
paginate: true
---

# Codebase Explanation: ChatGPT CLI

A command-line interface for interactive chat with Large Language Models (LLMs) from OpenAI and Google Vertex AI.

---

## Overview

*   **Purpose**: Provides an interactive chat experience directly from the terminal.
*   **LLM Integration**: Supports both OpenAI (ChatGPT) and Google Vertex AI (Gemini) models.
*   **User Interface**: Uses `liner` library for an enhanced command-line input experience (e.g., history, Ctrl+D to exit).

---

## Key Components: `main.go`

The entire application logic resides in `main.go`.

### Data Structures

*   `Message`: Represents a single message in the conversation (`role`, `content`).
*   `ChatCompletionRequest`: Payload for OpenAI API requests.
*   `ChatCompletionChoice`: A single choice from OpenAI API response.
*   `ChatCompletionResponse`: Full response from OpenAI API.

---

## Key Components: Functions

*   `sendChat(apiKey, messages, model)`:
    *   Handles communication with the OpenAI Chat Completion API.
    *   Constructs the request, sets headers (including API key), and parses the response.
    *   Returns the assistant's reply or an error.

---

## Key Components: Functions (cont.)

*   `isVertexModel(model string)`:
    *   Simple helper function to check if the provided `model` string indicates a Google Vertex AI Gemini model (e.g., starts with "gemini").

*   `getReply(apiKey, messages, model)`:
    *   Acts as a dispatcher.
    *   If `isVertexModel` returns true, it calls `sendVertexChat`.
    *   Otherwise, it calls `sendChat` for OpenAI models.

---

## Key Components: Functions (cont.)

*   `sendVertexChat(messages, model)`:
    *   Handles communication with Google Vertex AI (Gemini) API.
    *   Requires `GOOGLE_CLOUD_PROJECT` and optionally `VERTEX_LOCATION` environment variables.
    *   Supports both API key (`VERTEX_API_KEY`) and Application Default Credentials (ADC) for authentication.
    *   Formats messages for the Vertex AI API and parses the response.

---

## Main Application Flow

```go
func main() {
    // 1. Get OPENAI_API_KEY from environment.
    // 2. Parse command-line flags: -model, -system.
    // 3. Initialize liner for interactive input.
    // 4. Handle initial system prompt (if provided).
    // 5. Enter an infinite loop for user interaction:
    //    a. Prompt user for input.
    //    b. Add user message to conversation history.
    //    c. Call getReply to get assistant's response.
    //    d. Display assistant's response.
    //    e. Add assistant message to history.
    //    f. Handle 'exit' command or Ctrl+D.
}
```

---

## How to Run

1.  **Compile**:
    ```bash
    go build -o q main.go
    ```
2.  **Set API Key**:
    ```bash
    export OPENAI_API_KEY="your_openai_api_key"
    # For Vertex AI:
    export GOOGLE_CLOUD_PROJECT="your_gcp_project_id"
    # export VERTEX_API_KEY="your_vertex_api_key" (optional, for API key auth)
    ```
3.  **Execute**:
    ```bash
    ./q -model gpt-4o-mini
    # Or for Gemini:
    ./q -model gemini-pro-1.0
    ```
