package market_test

import (
	"nofx/market"
	"testing"
	"time"
)

func TestComputeAnchors_SimplePivot(t *testing.T) {
	series := make(map[string]*market.TimeframeSeriesData)
	var bars []market.KlineBar
	base := time.Now().Add(-time.Hour * 24).UnixMilli()
	bars = append(bars, market.KlineBar{Time: base + 0*int64(time.Hour/time.Millisecond), High: 100, Low: 95})
	bars = append(bars, market.KlineBar{Time: base + 1*int64(time.Hour/time.Millisecond), High: 102, Low: 96})
	bars = append(bars, market.KlineBar{Time: base + 2*int64(time.Hour/time.Millisecond), High: 104, Low: 97})
	bars = append(bars, market.KlineBar{Time: base + 3*int64(time.Hour/time.Millisecond), High: 106, Low: 98})
	bars = append(bars, market.KlineBar{Time: base + 4*int64(time.Hour/time.Millisecond), High: 97, Low: 85})
	bars = append(bars, market.KlineBar{Time: base + 5*int64(time.Hour/time.Millisecond), High: 99, Low: 90})
	bars = append(bars, market.KlineBar{Time: base + 6*int64(time.Hour/time.Millisecond), High: 101, Low: 92})
	bars = append(bars, market.KlineBar{Time: base + 7*int64(time.Hour/time.Millisecond), High: 103, Low: 93})
	bars = append(bars, market.KlineBar{Time: base + 8*int64(time.Hour/time.Millisecond), High: 105, Low: 94})
	bars = append(bars, market.KlineBar{Time: base + 9*int64(time.Hour/time.Millisecond), High: 107, Low: 96})
	series["1h"] = &market.TimeframeSeriesData{Timeframe: "1h", Klines: bars}
	anchors := market.ComputeAnchors(series)
	if len(anchors) == 0 {
		t.Fatalf("expected anchors, got none")
	}
}
