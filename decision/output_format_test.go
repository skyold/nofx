package decision

import (
	"nofx/store"
	"strings"
	"testing"
)

func TestBuildSystemPromptStrictOutputFormat(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(10000, "")

	if !strings.Contains(prompt, "【最终输出格式】") {
		t.Fatalf("prompt missing strict header")
	}
	if !strings.Contains(prompt, "<reasoning>") || !strings.Contains(prompt, "</reasoning>") {
		t.Fatalf("prompt missing reasoning tag block")
	}
	if !strings.Contains(prompt, "<decision>") || !strings.Contains(prompt, "</decision>") {
		t.Fatalf("prompt missing decision tag block")
	}
	if strings.Contains(prompt, "```json") {
		t.Fatalf("prompt should not include code fences in strict format")
	}
}
