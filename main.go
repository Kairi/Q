package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
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

// sendChat sends the conversation history to the OpenAI API and returns the assistant's reply
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

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "Error: OPENAI_API_KEY environment variable not set")
        os.Exit(1)
    }

   model := flag.String("model", "o4-mini", "model to use (e.g., gpt-4o-mini or gpt-4)")
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
       resp, err := sendChat(apiKey, messages, *model)
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
        resp, err := sendChat(apiKey, messages, *model)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
            continue
        }
        // display assistant response with colored label
        fmt.Printf("%s\U0001F916 ChatGPT:%s %s\n\n", ansiBlue, ansiReset, resp)
        messages = append(messages, Message{Role: "assistant", Content: resp})
    }
}
