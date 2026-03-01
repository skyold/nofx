// =============================================================================
// Chaos Trading System - User Prompt V2 (Text Format with Institutional Regime)
// =============================================================================
//
// V2 = EnhancedBuilder + TextFormatter (包含机构分类器 + LLM 简报)
// =============================================================================

package chaos

import (
	"strings"

	"nofx/market"
	"nofx/store"
)

// buildUserPromptV2 使用 ChaosContext 构建用户提示词（V2 版本）
// V2 = EnhancedBuilder + TextFormatter (包含机构分类器 + LLM 简报)
func (m *Manager) buildUserPromptV2(ctx *ChaosContext) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder

	// 1. 标题
	sb.WriteString("# 🌀 Chaos 模式用户提示 (V2 - Enhanced Text Format)\n\n")

	// 2. 公共信息部分
	sb.WriteString(m.buildHeader(ctx))
	sb.WriteString(m.buildGlobalContext(ctx))
	sb.WriteString(m.buildAccountStatus(ctx))
	sb.WriteString(m.buildTradingPerformance(ctx))
	sb.WriteString(m.buildPositions(ctx))

	// 3. 市场数据 (文本格式)
	sb.WriteString(m.buildCandidates(ctx))
	sb.WriteString(m.buildRankings(ctx))

	sb.WriteString("---\n\n")

	return sb.String()
}

// formatMarketDataV2 使用 EnhancedBuilder + TextFormatter 格式化市场数据
func (m *Manager) formatMarketDataV2(data *market.Data, indicators store.IndicatorConfig) string {
	// Builder: 构建增强数据（包含机构分类器 + LLM 简报）
	builder := &EnhancedBuilder{
		BasicBuilder: &BasicBuilder{Manager: m},
	}
	promptData := builder.BuildMarketData(data.Symbol, data, indicators)

	// Formatter: 格式化为文本（包含机构分类器和 LLM 简报）
	formatter := &TextFormatter{
		IncludeInstitutionalRegime: true,
		IncludeLLMBriefing:         true,
	}
	return formatter.FormatMarketData(promptData, indicators)
}
