// =============================================================================
// Chaos Trading System - User Prompt V1 (Legacy - Text Format)
// =============================================================================
//
// V1 = BasicBuilder + TextFormatter (不包含机构分类器)
// =============================================================================

package chaos

import (
	"strings"

	"nofx/market"
	"nofx/store"
)

// buildUserPromptV1 使用 ChaosContext 构建用户提示词（V1 版本）
// V1 = BasicBuilder + TextFormatter (不包含机构分类器)
func (m *Manager) buildUserPromptV1(ctx *ChaosContext) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder

	// 1. 标题
	sb.WriteString("# 🌀 Chaos 模式用户提示 (V1 - Legacy Text Format)\n\n")

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

// formatMarketDataV1 使用 BasicBuilder + TextFormatter 格式化市场数据
func (m *Manager) formatMarketDataV1(data *market.Data, indicators store.IndicatorConfig) string {
	// Builder: 构建基础数据（不包含机构分类器）
	builder := &BasicBuilder{Manager: m}
	promptData := builder.BuildMarketData(data.Symbol, data, indicators)

	// Formatter: 格式化为文本（不包含机构分类器和 LLM 简报）
	formatter := &TextFormatter{
		IncludeInstitutionalRegime: false,
		IncludeLLMBriefing:         false,
	}
	result := formatter.FormatMarketData(promptData, indicators)
	return result
}
