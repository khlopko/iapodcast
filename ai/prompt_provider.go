package ai

type PromptProvider interface {
	SystemPrompt() string
	UserPrompt() string
	String() string
}

