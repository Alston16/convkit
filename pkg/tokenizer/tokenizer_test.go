package tokenizer_test

import (
	"testing"

	"github.com/Alston16/convkit/pkg/tokenizer"
)

const testText = "The quick brown fox jumps over the lazy dog."

func TestCharEstimateFallback_NonZero(t *testing.T) {
	tok := tokenizer.New("unknown-model-xyz")
	count := tok.CountTokens("unknown-model-xyz", testText)
	if count <= 0 {
		t.Fatalf("expected positive token count, got %d", count)
	}
}

func TestTiktoken_GPT4_ReasonableCount(t *testing.T) {
	tok := tokenizer.New("gpt-4")
	count := tok.CountTokens("gpt-4", testText)
	if count <= 0 {
		t.Fatalf("expected positive token count, got %d", count)
	}
	// The sentence is ~44 chars. At ~4 chars/token that's ~11 tokens.
	// tiktoken should produce something close. Assert within ±5 tokens of 11.
	const expected = 11
	const tolerance = 5
	diff := count - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Fatalf("token count %d is too far from expected ~%d (tolerance ±%d)", count, expected, tolerance)
	}
}

func TestCharEstimate_SingleChar(t *testing.T) {
	tok := tokenizer.New("some-other-model")
	count := tok.CountTokens("some-other-model", "x")
	if count != 1 {
		t.Fatalf("expected 1 for single char, got %d", count)
	}
}

func TestCharEstimate_EmptyString(t *testing.T) {
	tok := tokenizer.New("some-other-model")
	count := tok.CountTokens("some-other-model", "")
	if count != 0 {
		t.Fatalf("expected 0 for empty string, got %d", count)
	}
}
