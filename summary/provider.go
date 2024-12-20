package summary

import (
	"errors"
)

type SummaryProvider interface {
	Prepare() error
	GenerateFromInput(input string) (string, error)
	String() string
}

type SummaryProviderErrorCode int

var (
	ErrFailedPreparation = errors.New("provider has failed to prepare")
)

type PromptProvider interface {
	SystemPrompt() string
	UserPrompt() string
	String() string
}

func NewSummaryProvider(name string, promptProvider PromptProvider) SummaryProvider {
	if name == "openai" {
		return &OpenAiSummaryProvider{promptProvider, nil}
	}
	if name == "claude" {
		return &AnthropicSummaryProvider{promptProvider, nil}
	}
	return nil
}

