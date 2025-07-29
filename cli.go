package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peterh/liner"
)

// CLIHandler manages the command-line interface interactions
type CLIHandler struct {
	liner      *liner.State
	model      string
	ansiColors map[string]string
}

// NewCLIHandler creates a new CLI handler with initialized components
func NewCLIHandler(model string) *CLIHandler {
	rl := liner.NewLiner()
	rl.SetCtrlCAborts(true)
	rl.SetMultiLineMode(true)

	return &CLIHandler{
		liner: rl,
		model: model,
		ansiColors: map[string]string{
			"reset":  "\033[0m",
			"green":  "\033[32m",
			"blue":   "\033[34m",
			"yellow": "\033[33m",
		},
	}
}

// Close properly closes the CLI handler
func (c *CLIHandler) Close() {
	c.liner.Close()
}

// PrintHeader displays the application header
func (c *CLIHandler) PrintHeader() {
	fmt.Printf("%s%s interactive chat (%s)%s\n", 
		c.ansiColors["yellow"], AppName, c.model, c.ansiColors["reset"])
}

// HandleInitialCommands handles the initial command selection (/new, /load, /list)
func (c *CLIHandler) HandleInitialCommands() ([]Message, string, error) {
	threads, err := listConversations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing conversations: %v\n", err)
	}

	c.displayAvailableThreads(threads)

	for {
		fmt.Print(c.ansiColors["green"])
		line, err := c.liner.Prompt("Command (e.g., /new, /load <name>, /list): ")
		fmt.Print(c.ansiColors["reset"])
		
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nExiting.")
				return nil, "", err
			}
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "/load ") {
			return c.handleLoadCommand(line)
		} else if line == "/new" {
			return c.handleNewCommand()
		} else if line == "/list" {
			c.handleListCommand()
		} else {
			fmt.Println("Invalid command. Use '/new', '/load <name>', or '/list'.")
		}
	}
}

// displayAvailableThreads shows existing conversations to the user
func (c *CLIHandler) displayAvailableThreads(threads []string) {
	if len(threads) > 0 {
		fmt.Println("Existing conversations:")
		for _, t := range threads {
			fmt.Printf("- %s\n", t)
		}
		fmt.Println("\nType '/load <name>' to load a conversation, or '/new' to start a new one.")
	} else {
		fmt.Println("No existing conversations. Type '/new' to start a new one.")
	}
}

// handleLoadCommand handles loading an existing conversation
func (c *CLIHandler) handleLoadCommand(line string) ([]Message, string, error) {
	name := strings.TrimPrefix(line, "/load ")
	loadedMessages, err := loadConversation(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading conversation '%s': %v\n", name, err)
		return nil, "", err
	}
	fmt.Printf("Conversation '%s' loaded. Type your message and press Ctrl+D to send. Type 'exit' to quit.\n", name)
	return loadedMessages, name, nil
}

// handleNewCommand handles creating a new conversation
func (c *CLIHandler) handleNewCommand() ([]Message, string, error) {
	for {
		fmt.Print(c.ansiColors["green"])
		name, err := c.liner.Prompt("Enter a name for the new conversation: ")
		fmt.Print(c.ansiColors["reset"])
		
		if err != nil {
			if err == io.EOF || err == liner.ErrPromptAborted {
				return nil, "", err
			}
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			continue
		}
		
		threadName := strings.TrimSpace(name)
		if threadName != "" {
			fmt.Printf("New conversation '%s' started. Type your message and press Ctrl+D to send. Type 'exit' to quit.\n", threadName)
			return []Message{}, threadName, nil
		}
		fmt.Println("Conversation name cannot be empty.")
	}
}

// handleListCommand handles listing all conversations
func (c *CLIHandler) handleListCommand() {
	threads, err := listConversations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing conversations: %v\n", err)
		return
	}
	if len(threads) == 0 {
		fmt.Println("No existing conversations.")
	} else {
		fmt.Println("Existing conversations:")
		for _, t := range threads {
			fmt.Printf("- %s\n", t)
		}
	}
}

// GetUserInput handles multi-line user input with proper exit handling
func (c *CLIHandler) GetUserInput(threadName string) (string, bool, error) {
	var inputBuilder strings.Builder
	
	fmt.Print(c.ansiColors["green"])
	for {
		line, err := c.liner.Prompt(fmt.Sprintf("[%s] You: ", threadName))
		fmt.Print(c.ansiColors["reset"])
		
		if err != nil {
			if err == liner.ErrPromptAborted {
				inputBuilder.Reset()
				break
			}
			if err == io.EOF {
				if inputBuilder.Len() == 0 {
					fmt.Println("\nExiting.")
					return "", true, nil
				}
				break
			}
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			inputBuilder.Reset()
			break
		}

		if line == "exit" {
			return "exit", true, nil
		}

		if inputBuilder.Len() > 0 {
			inputBuilder.WriteString("\n")
		}
		inputBuilder.WriteString(line)
	}

	input := strings.TrimSpace(inputBuilder.String())
	return input, false, nil
}

// HandleExitSave handles the save prompt when exiting
func (c *CLIHandler) HandleExitSave(messages []Message, threadName string) error {
	if threadName == "" {
		return nil
	}
	
	fmt.Print(c.ansiColors["green"])
	savePrompt, err := c.liner.Prompt(fmt.Sprintf("Save conversation '%s'? (yes/no): ", threadName))
	fmt.Print(c.ansiColors["reset"])
	
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	
	if strings.ToLower(strings.TrimSpace(savePrompt)) == "yes" {
		if err := saveConversation(messages, threadName); err != nil {
			return fmt.Errorf("error saving conversation: %w", err)
		}
		fmt.Println("Conversation saved.")
	}
	return nil
}

// AddToHistory adds user input to command history
func (c *CLIHandler) AddToHistory(input string) {
	c.liner.AppendHistory(input)
}

// PrintThinking displays the model thinking message
func (c *CLIHandler) PrintThinking() {
	fmt.Printf("%s is thinking...\n", c.model)
}

// PrintResponse displays the assistant's response with colored formatting
func (c *CLIHandler) PrintResponse(response string) {
	fmt.Printf("%s🤖 ChatGPT:%s %s\n\n", 
		c.ansiColors["blue"], c.ansiColors["reset"], response)
}

// PrintSystemPrompt displays the system prompt message
func (c *CLIHandler) PrintSystemPrompt(prompt string) {
	fmt.Printf("System prompt: %s\n\n", prompt)
}