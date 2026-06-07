package tokenizer

import (
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// Tokenizer counts tokens for a given model. Implementations are selected at
// runtime by model ID so that callers never hardcode a tokeniser.
type Tokenizer interface {
	CountTokens(model, text string) int
}

// factory is the top-level implementation that dispatches to the correct
// per-model tokeniser.
type factory struct{}

// New returns the Tokenizer appropriate for modelID. If the model is an
// OpenAI model supported by tiktoken-go, the tiktoken implementation is used.
// All other models fall back to a character-estimate approximation.
func New(modelID string) Tokenizer {
	return &factory{}
}

func (f *factory) CountTokens(model, text string) int {
	if isTiktokenModel(model) {
		return countTiktoken(model, text)
	}
	return countCharEstimate(text)
}

// isTiktokenModel returns true for OpenAI model IDs that tiktoken-go supports.
func isTiktokenModel(model string) bool {
	known := []string{
		"gpt-4", "gpt-4o", "gpt-3.5-turbo", "gpt-3.5",
		"text-embedding-ada-002", "text-davinci",
	}
	for _, prefix := range known {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// countTiktoken uses tiktoken-go to produce an exact token count.
// Falls back to charEstimate if the encoding cannot be loaded.
func countTiktoken(model, text string) int {
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return countCharEstimate(text)
	}
	return len(enc.Encode(text, nil, nil))
}

// countCharEstimate is a model-agnostic approximation: ~4 characters per token.
func countCharEstimate(text string) int {
	n := len(text) / 4
	if n == 0 && len(text) > 0 {
		return 1
	}
	return n
}
