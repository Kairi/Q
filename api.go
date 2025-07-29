package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func sendChat(apiKey string, messages []Message, model string) (string, error) {
	reqBody := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoints := DefaultAPIEndpoints()
	req, err := http.NewRequest("POST", endpoints.OpenAI, bytes.NewBuffer(bodyBytes))
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
	apiKey := os.Getenv(EnvOpenAIKey)
	if apiKey == "" {
		return "", fmt.Errorf("%s environment variable not set for OpenAI model", EnvOpenAIKey)
	}
	return sendChat(apiKey, messages, model)
}

// sendVertexChat sends conversation history to Google Gemini API and returns the assistant's reply
func sendVertexChat(messages []Message, model string) (string, error) {
	apiKey := os.Getenv(EnvGeminiKey)
	if apiKey == "" {
		return "", fmt.Errorf("%s environment variable not set", EnvGeminiKey)
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
