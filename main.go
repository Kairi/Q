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
	"path/filepath"
	"strings"

	// external libraries
	"github.com/google/generative-ai-go/genai"
	"github.com/peterh/liner"
	"google.golang.org/api/option"
)

// Message represents a single message in the chat conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest is the payload sent to the OpenAI chat completion API
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
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
		Model:    model,
		Messages: messages,
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
	cs := gm.StartChat()

	var systemPrompt string
	// Handle system message if present. It must be the first message.
	if len(messages) > 0 && messages[0].Role == "system" {
		systemPrompt = messages[0].Content
		messages = messages[1:]
	}

	if systemPrompt != "" {
		gm.SystemInstruction = &genai.Content{Parts: []genai.Part{genai.Text(systemPrompt)}}
	}

	// Add previous messages to history
	cs.History = make([]*genai.Content, 0, len(messages)-1)
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

	model := flag.String("model", "gemini-2.5-flash-lite-preview-06-17", "model to use (e.g., gpt-4o-mini, gpt-4, or Gemini model like gemini-pro-1.0, gemini-2.5-flash-lite-preview-06-17)")
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
	var threadName string
	// Check for existing conversations
	threads, err := listConversations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing conversations: %v\n", err)
	}

	// Initialize Emacs-style line editor
	rl := liner.NewLiner()
	defer rl.Close()
	rl.SetCtrlCAborts(true)
	rl.SetMultiLineMode(true)

	if len(threads) > 0 {
		fmt.Println("Existing conversations:")
		for _, t := range threads {
			fmt.Printf("- %s\n", t)
		}
		fmt.Println("\nType '/load <name>' to load a conversation, or '/new' to start a new one.")
	} else {
		fmt.Println("No existing conversations. Type '/new' to start a new one.")
	}

	// Initialize chat session
	for {
		fmt.Print(ansiGreen)
		line, err := rl.Prompt("Command (e.g., /new, /load <name>, /list): ")
		fmt.Print(ansiReset)
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nExiting.")
				return
			}
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/load ") {
			name := strings.TrimPrefix(line, "/load ")
			loadedMessages, err := loadConversation(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading conversation '%s': %v\n", name, err)
				continue
			}
			messages = loadedMessages
			threadName = name
			fmt.Printf("Conversation '%s' loaded. Type your message and press Ctrl+D to send. Type 'exit' to quit.\n", threadName)
			break
		} else if line == "/new" {
			for {
				fmt.Print(ansiGreen)
				name, err := rl.Prompt("Enter a name for the new conversation: ")
				fmt.Print(ansiReset)
				if err != nil {
					if err == io.EOF || err == liner.ErrPromptAborted {
						threadName = ""
						break // User cancelled
					}
					fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
					continue
				}
				threadName = strings.TrimSpace(name)
				if threadName != "" {
					break // Valid name, break inner loop
				}
				fmt.Println("Conversation name cannot be empty.")
			}
			if threadName == "" { // If user cancelled creating a new thread
				continue
			}
			messages = []Message{} // Start a new empty conversation
			fmt.Printf("New conversation '%s' started. Type your message and press Ctrl+D to send. Type 'exit' to quit.\n", threadName)
			break
		} else if line == "/list" {
			threads, err := listConversations()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing conversations: %v\n", err)
				continue
			}
			if len(threads) == 0 {
				fmt.Println("No existing conversations.")
			} else {
				fmt.Println("Existing conversations:")
				for _, t := range threads {
					fmt.Printf("- %s\n", t)
				}
			}
		} else {
			fmt.Println("Invalid command. Use '/new', '/load <name>', or '/list'.")
		}
	}

	// Only apply system prompt if it's a new conversation and the prompt is provided
	if len(messages) == 0 && *system != "" {
		messages = append(messages, Message{Role: "system", Content: *system})
		fmt.Printf("System prompt: %s\n\n", *system)
		// send initial system prompt to get assistant's response
		fmt.Printf("%s is thinking...\n", *model)
		resp, err := getReply(messages, *model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
		} else {
			fmt.Printf("%s\U0001F916 ChatGPT:%s %s\n\n", ansiBlue, ansiReset, resp)
			messages = append(messages, Message{Role: "assistant", Content: resp})
		}
	}
	for {
		var inputBuilder strings.Builder
		fmt.Print(ansiGreen)
		for {
			// Include current thread name in the prompt
			line, err := rl.Prompt(fmt.Sprintf("[%s] You: ", threadName))
			fmt.Print(ansiReset) // Reset color after prompt
			if err != nil {
				if err == liner.ErrPromptAborted {
					inputBuilder.Reset() // Clear buffer if prompt aborted
					break                // Break inner loop to re-prompt
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
				break                // Break inner loop to re-prompt
			}

			if line == "exit" {
				if threadName != "" {
					fmt.Print(ansiGreen)
					savePrompt, err := rl.Prompt(fmt.Sprintf("Save conversation '%s'? (yes/no): ", threadName))
					fmt.Print(ansiReset)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
					} else if strings.ToLower(strings.TrimSpace(savePrompt)) == "yes" {
						if err := saveConversation(messages, threadName); err != nil {
							fmt.Fprintf(os.Stderr, "Error saving conversation: %v\n", err)
						} else {
							fmt.Println("Conversation saved.")
						}
					}
				}
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
		fmt.Printf("%s is thinking...\n", *model)
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

// getHistoryDir ensures the history directory exists and returns its path.
func getHistoryDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	historyDir := filepath.Join(configDir, "q", "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create history directory: %w", err)
	}
	return historyDir, nil
}

// saveConversation saves the conversation history to a file in the user's config directory.
func saveConversation(messages []Message, threadName string) error {
	historyDir, err := getHistoryDir()
	if err != nil {
		return err
	}
	filePath := filepath.Join(historyDir, fmt.Sprintf("%s.json", threadName))
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create conversation file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(messages); err != nil {
		return fmt.Errorf("failed to encode conversation: %w", err)
	}
	return nil
}

// loadConversation loads the conversation history from a file in the user's config directory.
func loadConversation(threadName string) ([]Message, error) {
	historyDir, err := getHistoryDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(historyDir, fmt.Sprintf("%s.json", threadName))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open conversation file: %w", err)
	}
	defer file.Close()

	var messages []Message
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&messages); err != nil {
		return nil, fmt.Errorf("failed to decode conversation: %w", err)
	}
	return messages, nil
}

// listConversations lists all available conversation threads from the user's config directory.
func listConversations() ([]string, error) {
	historyDir, err := getHistoryDir()
	if err != nil {
		// If the config directory itself can't be found/created, it's a problem.
		return nil, err
	}

	files, err := os.ReadDir(historyDir)
	if err != nil {
		// If we can't read the directory (e.g., permissions), that's an error.
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	var threads []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			threads = append(threads, strings.TrimSuffix(file.Name(), ".json"))
		}
	}
	return threads, nil
}

