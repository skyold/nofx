// =============================================================================
// Chaos Trading System - Prompt Helpers
// =============================================================================
//
// 辅助函数和工具函数
// 供 prompt_user.go 和 prompt_system.go 使用
//
// 注意：以下函数已在其他文件中定义，不要重复定义：
// - formatPriceForPrompt, formatFloatSlice (utils.go)
// - SwingType, SwingPoint, TechnicalFeatures, GenerateTechnicalFeatures, countLevelTests (signal.go)
// =============================================================================

package chaos

import (
	"math"
	"sort"
)

// ============================================================================
// 常量定义
// ============================================================================

const (
	maxSwingFeatures = 5
	timeFormatUTC    = "01-02 15:04"
	timeFormatTime   = "15:04"
)

// ============================================================================
// 工具函数 - 数值计算
// ============================================================================

// roundFloat 四舍五入到指定精度
func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// calculatePercentile 计算百分位
func calculatePercentile(value float64, history []float64) int {
	if len(history) == 0 {
		return 50
	}

	sorted := make([]float64, len(history))
	copy(sorted, history)
	sort.Float64s(sorted)

	count := 0
	for _, v := range sorted {
		if v <= value {
			count++
		}
	}

	percentile := float64(count) / float64(len(sorted)) * 100
	return int(math.Round(percentile))
}
