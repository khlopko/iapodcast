package summary

import (
	"errors"
)

type SummaryProvider interface {
	Prepare() error
	GenerateFromInput(input string) (string, error)
}

type SummaryProviderErrorCode int

var (
	ErrFailedPreparation = errors.New("provider has failed to prepare")
)

func NewSummaryProvider(name string) SummaryProvider {
	if name == "openai" {
		return &OpenAiSummaryProvider{}
	}
	if name == "claude" {
		return &AnthropicSummaryProvider{}
	}
	return nil
}

