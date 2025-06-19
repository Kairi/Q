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
  


   // external libraries
   "github.com/peterh/liner"
   "github.com/google/generative-ai-go/genai"
   "google.golang.org/api/option"
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
func getReply(messages []Message, model string) (string, error) {
   if isVertexModel(model) {
       return sendVertexChat(messages, model)
   }
   apiKey := os.Getenv("OPENAI_API_KEY")
   if apiKey == "" {
       return "", fmt.Errorf("OPENAI_API_KEY environment variable not set for OpenAI model")
   }
   return sendChat(apiKey, messages, model)
}

// sendVertexChat sends conversation history to Google Gemini API and returns the assistant's reply
func sendVertexChat(messages []Message, model string) (string, error) {
   apiKey := os.Getenv("GEMINI_API_KEY")
   if apiKey == "" {
       return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
   }

   ctx := context.Background()
   client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
   if err != nil {
       return "", fmt.Errorf("failed to create Gemini client: %w", err)
   }
   defer client.Close()

   gm := client.GenerativeModel(model)
   
   var cs *genai.ChatSession
   
   // Handle system message if present
   if len(messages) > 0 && messages[0].Role == "system" {
       cs = gm.StartChat()
       _, err := cs.SendMessage(ctx, genai.Text(messages[0].Content))
       if err != nil {
           return "", fmt.Errorf("failed to send initial system message to Gemini: %w", err)
       }
       messages = messages[1:] // Remove system message
   } else {
       cs = gm.StartChat()
   }

   // Add previous messages to history
   for _, msg := range messages[:len(messages)-1] { // All messages except the last one
       var role string
       if msg.Role == "user" {
           role = "user"
       } else if msg.Role == "assistant" {
           role = "model"
       } else {
           continue // Skip unknown roles
       }
       cs.History = append(cs.History, &genai.Content{
           Role:  role,
           Parts: []genai.Part{genai.Text(msg.Content)},
       })
   }

   // Send the last message
   resp, err := cs.SendMessage(ctx, genai.Text(messages[len(messages)-1].Content))
   if err != nil {
       return "", fmt.Errorf("failed to send message to Gemini: %w", err)
   }

   if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
       return "", fmt.Errorf("no candidates in Gemini response")
   }

   return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
}

func main() {

   model := flag.String("model", "gemini-2.5-flash-lite", "model to use (e.g., gpt-4o-mini, gpt-4, or Gemini model like gemini-pro-1.0, gemini-2.5-flash-lite)")
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
       resp, err := getReply(messages, *model)
       if err != nil {
           fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
       } else {
           fmt.Printf("%s\U0001F916 ChatGPT:%s %s\n\n", ansiBlue, ansiReset, resp)
           messages = append(messages, Message{Role: "assistant", Content: resp})
       }
   }
   fmt.Println("Type your message and press Ctrl+D to send. Type 'exit' to quit.")
   // initialize Emacs-style line editor
   rl := liner.NewLiner()
   defer rl.Close()
   rl.SetCtrlCAborts(true)
   rl.SetMultiLineMode(true)

    for {
        var inputBuilder strings.Builder
        fmt.Print(ansiGreen)
        for {
            line, err := rl.Prompt("You: ")
            fmt.Print(ansiReset) // Reset color after prompt
            if err != nil {
                if err == liner.ErrPromptAborted {
                    inputBuilder.Reset() // Clear buffer if prompt aborted
                    break // Break inner loop to re-prompt
                }
                if err == io.EOF {
                    // EOF means end of input, send the message
                    if inputBuilder.Len() == 0 {
                        fmt.Println("\nExiting.")
                        return // Exit if EOF on empty input
                    }
                    break // Break inner loop to process accumulated input
                }
                fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
                inputBuilder.Reset() // Clear buffer on error
                break // Break inner loop to re-prompt
            }

            if line == "exit" {
                fmt.Println("Exiting.")
                return
            }

            if inputBuilder.Len() > 0 {
                inputBuilder.WriteString("\n") // Add newline for multi-line input
            }
            inputBuilder.WriteString(line)
        }

        input := strings.TrimSpace(inputBuilder.String())
        if input == "" {
            continue
        }

        rl.AppendHistory(input)

        messages = append(messages, Message{Role: "user", Content: input})
        fmt.Println("ChatGPT is thinking...")
        resp, err := getReply(messages, *model)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
            continue
        }
        // display assistant response with colored label
        fmt.Printf("%s\U0001F916 ChatGPT:%s %s\n\n", ansiBlue, ansiReset, resp)
        messages = append(messages, Message{Role: "assistant", Content: resp})
    }
}
