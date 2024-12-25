package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type OpenAIRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type OpenAiServiceProvider struct {
	promptProvider PromptProvider
	apiKey *string
}

func (self *OpenAiServiceProvider) Prepare() error {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		return ErrFailedPreparation
	}
	self.apiKey = &openAIKey
	return nil
}

func (self *OpenAiServiceProvider) GenerateFromInput(input string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: self.promptProvider.SystemPrompt(),
			},
			{
				Role:    "user",
				Content: fmt.Sprintf(self.promptProvider.UserPrompt(), input),
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+*self.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no summary generated")
	}

	return response.Choices[0].Message.Content, nil
}

func (self *OpenAiServiceProvider) String() string {
	return fmt.Sprintf("openai-%s", self.promptProvider.String())
}
