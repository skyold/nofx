package chaos

import (
	"fmt"
	"testing"

	"nofx/market"
	"nofx/store"
)

// TestV2IncludesInstitutionalRegime 测试 V2 版本是否包含机构分类器
func TestV2IncludesInstitutionalRegime(t *testing.T) {
	// 创建模拟数据
	data := &market.Data{
		Symbol:       "ETHUSDT",
		CurrentPrice: 2000.0,
		FundingRate:  0.0001,
		OpenInterest: &market.OIData{
			Latest:  1000000.0,
			Average: 950000.0,
		},
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"1h": {
				Timeframe: "1h",
				Klines: []market.KlineBar{
					{Close: 1900.0, High: 1910.0, Low: 1890.0},
					{Close: 1920.0, High: 1930.0, Low: 1910.0},
					{Close: 1940.0, High: 1950.0, Low: 1930.0},
					{Close: 1960.0, High: 1970.0, Low: 1950.0},
					{Close: 1980.0, High: 1990.0, Low: 1970.0},
					{Close: 2000.0, High: 2010.0, Low: 1990.0},
				},
				EMA20Values: []float64{1950.0, 1955.0, 1960.0, 1965.0, 1970.0, 1975.0},
				EMA50Values: []float64{1900.0, 1905.0, 1910.0, 1915.0, 1920.0, 1925.0},
				ATR14Values: []float64{20.0, 21.0, 22.0, 23.0, 24.0, 25.0},
			},
		},
	}

	indicators := store.IndicatorConfig{
		EnableEMA:         true,
		EnableMACD:        false,
		EnableRSI:         false,
		EnableOI:          true,
		EnableFundingRate: true,
	}

	// 测试 V1
	m := &Manager{}
	v1Output := m.formatMarketDataV1(data, indicators)

	// 测试 V2
	v2Output := m.formatMarketDataV2(data, indicators)

	fmt.Println("=== V1 Output ===")
	fmt.Println(v1Output)
	fmt.Println("\n=== V2 Output ===")
	fmt.Println(v2Output)

	// 验证 V1 不包含机构分类器
	if contains(v1Output, "Institutional Regime Classifier") {
		t.Error("V1 should NOT contain Institutional Regime Classifier")
	}

	// 验证 V2 包含机构分类器
	if !contains(v2Output, "Institutional Regime Classifier") {
		t.Error("V2 SHOULD contain Institutional Regime Classifier")
	}

	// 验证 V2 包含 LLM 战术简报
	if !contains(v2Output, "市场战术简报") {
		t.Error("V2 SHOULD contain LLM Briefing (市场战术简报)")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
