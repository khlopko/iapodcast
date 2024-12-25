package ai

import "errors"

type AiServiceProvider interface {
	Prepare() error
	GenerateFromInput(input string) (string, error)
	String() string
}

type SummaryProviderErrorCode int

var (
	ErrFailedPreparation = errors.New("provider has failed to prepare")
)

type AiServiceType string

const (
	OpenAiServiceType AiServiceType = "openai"
	AnthropicServiceType AiServiceType = "anthropic"
)

func NewAiServiceProvider(serviceType AiServiceType, promptProvider PromptProvider) AiServiceProvider {
	switch serviceType {
	case OpenAiServiceType:
		return &OpenAiServiceProvider{promptProvider, nil}
	case AnthropicServiceType:
		return &AnthropicServiceProvider{promptProvider, nil}
	}

	return nil
}

