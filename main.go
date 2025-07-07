package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/peterh/liner"
)

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