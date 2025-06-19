package main

import (
   // standard libraries
   "bytes"
   "context"
   "encoding/json"
   "flag"
   "fmt"
   "io"
   "net/http"
   "os"
   "strings"
  
   "net/url"

   // external libraries
   "golang.org/x/oauth2"
   "golang.org/x/oauth2/google"
   "github.com/peterh/liner"
)

// Message represents a single message in the chat conversation
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// ChatCompletionRequest is the payload sent to the OpenAI chat completion API
type ChatCompletionRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
}

// ChatCompletionChoice represents a single choice returned by the API
type ChatCompletionChoice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason string  `json:"finish_reason"`
}

// ChatCompletionResponse is the response from the OpenAI chat completion API
type ChatCompletionResponse struct {
    ID      string                 `json:"id"`
    Object  string                 `json:"object"`
    Created int64                  `json:"created"`
    Model   string                 `json:"model"`
    Choices []ChatCompletionChoice `json:"choices"`
}

func sendChat(apiKey string, messages []Message, model string) (string, error) {
    reqBody := ChatCompletionRequest{
        Model:       model,
        Messages:    messages,
    }
    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)

    client := http.DefaultClient
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        respData, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("API error: %s", string(respData))
    }

    var respBody ChatCompletionResponse
    if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
        return "", err
    }
    if len(respBody.Choices) == 0 {
        return "", fmt.Errorf("no choices in response")
    }
    return respBody.Choices[0].Message.Content, nil
}
// isVertexModel returns true if the model name indicates a Google Vertex AI Gemini model
func isVertexModel(model string) bool {
   return strings.HasPrefix(model, "gemini")
}

// getReply dispatches the request to OpenAI or Vertex AI based on model prefix
func getReply(apiKey string, messages []Message, model string) (string, error) {
   if isVertexModel(model) {
       return sendVertexChat(messages, model)
   }
   return sendChat(apiKey, messages, model)
}

// sendVertexChat sends conversation history to Google Vertex AI (Gemini) and returns the assistant's reply
func sendVertexChat(messages []Message, model string) (string, error) {
   project := os.Getenv("GOOGLE_CLOUD_PROJECT")
   if project == "" {
       return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
   }
   location := os.Getenv("VERTEX_LOCATION")
   if location == "" {
       location = "us-central1"
   }
   // determine endpoint version: preview models require v1beta1
   version := "v1"
   if strings.Contains(model, "preview") {
       version = "v1beta1"
   }
   // build base endpoint URL
   path := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s:generateMessage",
       project, location, model)
   baseURL := fmt.Sprintf("https://%s-aiplatform.googleapis.com/%s/%s",
       location, version, path)
   // choose auth: API key or OAuth2
   var client *http.Client
   endpoint := baseURL
   if key := os.Getenv("VERTEX_API_KEY"); key != "" {
       // use API key auth
       endpoint = endpoint + "?key=" + url.QueryEscape(key)
       client = http.DefaultClient
   } else {
       // use Application Default Credentials
       ctx := context.Background()
       creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
       if err != nil {
           return "", fmt.Errorf("failed to obtain default credentials: %w", err)
       }
       client = oauth2.NewClient(ctx, creds.TokenSource)
   }

   // prepare request body
   type vertexMsg struct {
       Author  string `json:"author"`
       Content string `json:"content"`
   }
   type instance struct {
       Context  string      `json:"context,omitempty"`
       Messages []vertexMsg `json:"messages"`
   }
   var inst instance
   if len(messages) > 0 && messages[0].Role == "system" {
       inst.Context = messages[0].Content
       for _, m := range messages[1:] {
           if m.Role == "user" || m.Role == "assistant" {
               inst.Messages = append(inst.Messages, vertexMsg{Author: m.Role, Content: m.Content})
           }
       }
   } else {
       for _, m := range messages {
           if m.Role == "user" || m.Role == "assistant" {
               inst.Messages = append(inst.Messages, vertexMsg{Author: m.Role, Content: m.Content})
           }
       }
   }
   reqBody := struct {
       Instances  []instance            `json:"instances"`
       Parameters map[string]interface{} `json:"parameters,omitempty"`
   }{
       Instances: []instance{inst},
   }

   bodyBytes, err := json.Marshal(reqBody)
   if err != nil {
       return "", err
   }
   req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
   if err != nil {
       return "", err
   }
   req.Header.Set("Content-Type", "application/json")

   resp, err := client.Do(req)
   if err != nil {
       return "", err
   }
   defer resp.Body.Close()
   respData, _ := io.ReadAll(resp.Body)
   if resp.StatusCode < 200 || resp.StatusCode >= 300 {
       return "", fmt.Errorf("Vertex API error: %s", string(respData))
   }
   var respBody struct {
       Predictions []struct {
           Candidates []vertexMsg `json:"candidates"`
       } `json:"predictions"`
   }
   if err := json.Unmarshal(respData, &respBody); err != nil {
       return "", err
   }
   if len(respBody.Predictions) == 0 || len(respBody.Predictions[0].Candidates) == 0 {
       return "", fmt.Errorf("no candidates in response")
   }
   return respBody.Predictions[0].Candidates[0].Content, nil
}

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "Error: OPENAI_API_KEY environment variable not set")
        os.Exit(1)
    }

   model := flag.String("model", "o4-mini", "model to use (e.g., gpt-4o-mini, gpt-4, or Gemini model like gemini-pro-1.0)")
   system := flag.String("system", "", "optional initial system prompt to set assistant context")
   flag.Parse()

   // ANSI color codes for richer prompts
   const (
       ansiReset  = "\033[0m"
       ansiGreen  = "\033[32m"
       ansiBlue   = "\033[34m"
       ansiYellow = "\033[33m"
   )
   // Header
   fmt.Printf("%sChatGPT CLI interactive chat (%s)%s\n", ansiYellow, *model, ansiReset)
   var messages []Message
   if *system != "" {
       messages = append(messages, Message{Role: "system", Content: *system})
       fmt.Printf("System prompt: %s\n\n", *system)
       // send initial system prompt to get assistant's response
       fmt.Println("ChatGPT is thinking...")
       resp, err := getReply(apiKey, messages, *model)
       if err != nil {
           fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
       } else {
           fmt.Printf("%s\U0001F916 ChatGPT:%s %s\n\n", ansiBlue, ansiReset, resp)
           messages = append(messages, Message{Role: "assistant", Content: resp})
       }
   }
   fmt.Println("Type your message and press Enter. Type 'exit' or Ctrl+D to quit.")
   // initialize Emacs-style line editor
   rl := liner.NewLiner()
   defer rl.Close()
   rl.SetCtrlCAborts(true)

    for {
        // prompt with non-deletable prefix "You: ", colored green
        fmt.Print(ansiGreen)
        input, err := rl.Prompt("You: ")
        fmt.Print(ansiReset)
        if err != nil {
            if err == liner.ErrPromptAborted {
                continue
            }
            if err == io.EOF {
                fmt.Println("\nExiting.")
                return
            }
            fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
            continue
        }
        input = strings.TrimSpace(input)
        if input == "" {
            continue
        }
        if input == "exit" {
            fmt.Println("Exiting.")
            return
        }
        rl.AppendHistory(input)

        messages = append(messages, Message{Role: "user", Content: input})
        fmt.Println("ChatGPT is thinking...")
        resp, err := getReply(apiKey, messages, *model)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
            continue
        }
        // display assistant response with colored label
        fmt.Printf("%s\U0001F916 ChatGPT:%s %s\n\n", ansiBlue, ansiReset, resp)
        messages = append(messages, Message{Role: "assistant", Content: resp})
    }
}
