package main

// Config holds application configuration
type Config struct {
	Model  string
	System string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Model:  "gemini-2.5-flash-lite-preview-06-17",
		System: "",
	}
}

// APIEndpoints holds API endpoint configurations
type APIEndpoints struct {
	OpenAI string
}

// DefaultAPIEndpoints returns the default API endpoints
func DefaultAPIEndpoints() *APIEndpoints {
	return &APIEndpoints{
		OpenAI: "https://api.openai.com/v1/chat/completions",
	}
}

// Constants for the application
const (
	AppName       = "ChatGPT CLI"
	AppHistoryDir = "q"
	AppVersion    = "1.0.0"
)

// Environment variable names
const (
	EnvOpenAIKey = "OPENAI_API_KEY"
	EnvGeminiKey = "GEMINI_API_KEY"
)