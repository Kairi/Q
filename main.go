package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	model := flag.String("model", "gemini-2.5-flash-lite-preview-06-17", "model to use (e.g., gpt-5, gpt-4o-mini, gpt-4, or Gemini model like gemini-pro-1.0, gemini-2.5-flash-lite-preview-06-17)")
	system := flag.String("system", "", "optional initial system prompt to set assistant context")
	flag.Parse()

	cli := NewCLIHandler(*model)
	defer cli.Close()

	// Set up signal handling for graceful shutdown
	var messages []Message
	var threadName string
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Saving conversation...")
		if threadName != "" && len(messages) > 0 {
			if err := saveConversation(messages, threadName); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving conversation: %v\n", err)
			} else {
				fmt.Printf("Conversation '%s' saved.\n", threadName)
			}
		}
		fmt.Println("Exiting.")
		os.Exit(0)
	}()

	cli.PrintHeader()

	var err error
	messages, threadName, err = cli.HandleInitialCommands()
	if err != nil {
		return
	}

	// Only apply system prompt if it's a new conversation and the prompt is provided
	if len(messages) == 0 && *system != "" {
		messages = append(messages, Message{Role: "system", Content: *system})
		cli.PrintSystemPrompt(*system)
		// send initial system prompt to get assistant's response
		cli.PrintThinking()
		resp, err := getReply(messages, *model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
		} else {
			cli.PrintResponse(resp)
			messages = append(messages, Message{Role: "assistant", Content: resp})
		}
	}
	for {
		input, shouldExit, err := cli.GetUserInput(threadName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Input error: %v\n", err)
			continue
		}

		if shouldExit {
			if input == "exit" {
				if err := cli.HandleExitSave(messages, threadName); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				}
			}
			fmt.Println("Exiting.")
			return
		}

		if input == "" {
			continue
		}

		cli.AddToHistory(input)

		messages = append(messages, Message{Role: "user", Content: input})
		cli.PrintThinking()
		resp, err := getReply(messages, *model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
			continue
		}
		cli.PrintResponse(resp)
		messages = append(messages, Message{Role: "assistant", Content: resp})
	}
}