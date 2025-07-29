package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	model := flag.String("model", "gemini-2.5-flash-lite-preview-06-17", "model to use (e.g., gpt-4o-mini, gpt-4, or Gemini model like gemini-pro-1.0, gemini-2.5-flash-lite-preview-06-17)")
	system := flag.String("system", "", "optional initial system prompt to set assistant context")
	flag.Parse()

	cli := NewCLIHandler(*model)
	defer cli.Close()

	cli.PrintHeader()

	messages, threadName, err := cli.HandleInitialCommands()
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