package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
