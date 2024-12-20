package summary

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicSummaryProvider struct {
	promptProvider PromptProvider
	client *anthropic.Client
}

func NewAnthropicSummaryProvider(apiKey string) *AnthropicSummaryProvider {
	return &AnthropicSummaryProvider{}
}

func (self *AnthropicSummaryProvider) Prepare() error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return ErrFailedPreparation
	}

	self.client = anthropic.NewClient(option.WithAPIKey(apiKey))
	return nil
}

func (p *AnthropicSummaryProvider) GenerateFromInput(input string) (string, error) {
	if p.client == nil {
		return "", errors.New("client not initialized, call Prepare() first")
	}

	if input == "" {
		return "", errors.New("empty input provided")
	}

	msg := anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5HaikuLatest),
		MaxTokens: anthropic.Int(2048),
		System:    anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(p.promptProvider.SystemPrompt()),
		}),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf(p.promptProvider.UserPrompt(), input))),
		}),
	}

	ctx := context.Background()
	resp, err := p.client.Messages.New(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %v", err)
	}

	if len(resp.Content) == 0 {
		return "", errors.New("received empty response from API")
	}

	return resp.Content[0].Text, nil
}

func (self *AnthropicSummaryProvider) String() string {
	return fmt.Sprintf("anthropic-%s", self.promptProvider.String())
}

