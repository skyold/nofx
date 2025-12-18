package decision

import (
	"nofx/store"
	"strings"
	"testing"
)

func TestRiskConstraintsChaosOnlyCoreParts(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	engine := NewStrategyEngine(&cfg)
	p := engine.BuildSystemPrompt(5000, "")

	if !strings.Contains(p, "# 硬性约束（风险控制）") {
		t.Fatalf("missing risk constraints header")
	}
	if !strings.Contains(p, "最大持仓数") {
		t.Fatalf("missing max positions bullet")
	}
	if !strings.Contains(p, "仓位价值上限（山寨币）") || !strings.Contains(p, "仓位价值上限（BTC/ETH）") {
		t.Fatalf("missing position value limits bullets")
	}
	if !strings.Contains(p, "最大保证金使用率") {
		t.Fatalf("missing max margin usage bullet")
	}
	if !strings.Contains(p, "最小持仓规模") {
		t.Fatalf("missing min position size bullet")
	}
	if strings.Contains(p, "## AI 指引") {
		t.Fatalf("AI guidance should be removed in chaos mode")
	}
	if strings.Contains(p, "## 仓位规模指引") {
		t.Fatalf("position sizing guidance should be removed in chaos mode")
	}
}
