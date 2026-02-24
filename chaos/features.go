package chaos

import (
	"fmt"
	"strings"
	"time"

	"nofx/market"
)

// =============================================================================
// Feature Engineering: Swing Topology & Level Testing
// =============================================================================

type SwingType int

const (
	SwingHigh SwingType = iota
	SwingLow
)

type SwingPoint struct {
	Price     float64
	Time      int64     // Unix timestamp (ms)
	Index     int       // Index in the source Kline slice
	Type      SwingType // High or Low
	Label     string    // HH, LH, HL, LL
	TestCount int       // Number of times this level was tested
}

// TechnicalFeatures encapsulates the results of the feature engineering process
type TechnicalFeatures struct {
	SwingPoints []SwingPoint
	Trend       string // Simple trend description based on topology
}

// GenerateTechnicalFeatures performs the full feature extraction pipeline
func GenerateTechnicalFeatures(klines []market.KlineBar, window int) *TechnicalFeatures {
	// Need enough data for at least one window
	if len(klines) < window*2+1 {
		return &TechnicalFeatures{}
	}

	// 1. Identify Swing Points
	swings := findSwingPoints(klines, window)

	// 2. Label Topology (HH/HL/LH/LL)
	labelSwingTopology(swings)

	// 3. Count Level Tests
	// Use a tolerance of 0.2% (0.002) as requested
	tolerance := 0.002
	for i := range swings {
		// Count tests starting from the candle *after* the swing point
		startIndex := swings[i].Index + 1
		swings[i].TestCount = countLevelTests(swings[i].Price, swings[i].Type, klines, startIndex, tolerance)
	}

	return &TechnicalFeatures{
		SwingPoints: swings,
		Trend:       analyzeTrend(swings),
	}
}

// findSwingPoints identifies local highs and lows using a sliding window
func findSwingPoints(klines []market.KlineBar, window int) []SwingPoint {
	var swings []SwingPoint

	// We can only identify swings from [window] to [len-1-window]
	// Because we need N candles on both sides
	for i := window; i < len(klines)-window; i++ {
		isHigh := true
		isLow := true

		currentHigh := klines[i].High
		currentLow := klines[i].Low

		// Check left and right neighbors
		for j := i - window; j <= i+window; j++ {
			if i == j {
				continue
			}
			// Left side: Strict inequality
			if j < i {
				if klines[j].High > currentHigh {
					isHigh = false
				}
				if klines[j].Low < currentLow {
					isLow = false
				}
			} else {
				// Right side: Strict inequality + Equality (prioritize most recent peak)
				// If currentHigh == rightHigh, we fail current (so right one can be picked later)
				if klines[j].High >= currentHigh {
					isHigh = false
				}
				if klines[j].Low <= currentLow {
					isLow = false
				}
			}
		}

		// Add Swing High
		if isHigh {
			swings = append(swings, SwingPoint{
				Price: currentHigh,
				Time:  klines[i].Time,
				Index: i,
				Type:  SwingHigh,
			})
		}
		// Add Swing Low
		if isLow {
			swings = append(swings, SwingPoint{
				Price: currentLow,
				Time:  klines[i].Time,
				Index: i,
				Type:  SwingLow,
			})
		}
	}
	return swings
}

// labelSwingTopology determines HH/LH for highs and HL/LL for lows
func labelSwingTopology(swings []SwingPoint) {
	// Track the last seen High and Low separately
	var lastHigh *SwingPoint
	var lastLow *SwingPoint

	// Iterate through swings chronologically
	for i := 0; i < len(swings); i++ {
		// Use pointer to modify the element in the slice directly
		s := &swings[i]

		if s.Type == SwingHigh {
			if lastHigh == nil {
				s.Label = "High" // Initial High
			} else {
				if s.Price > lastHigh.Price {
					s.Label = "HH" // Higher High
				} else if s.Price < lastHigh.Price {
					s.Label = "LH" // Lower High
				} else {
					s.Label = "EH" // Equal High
				}
			}
			lastHigh = s
		} else { // SwingLow
			if lastLow == nil {
				s.Label = "Low" // Initial Low
			} else {
				if s.Price > lastLow.Price {
					s.Label = "HL" // Higher Low
				} else if s.Price < lastLow.Price {
					s.Label = "LL" // Lower Low
				} else {
					s.Label = "EL" // Equal Low
				}
			}
			lastLow = s
		}
	}
}

// countLevelTests counts how many times price tested a level within tolerance
// Based on user's logic:
// Support (Low): Price dips into [level-tol, level+tol] but closes > level+tol.
// Resistance (High): Price rallies into [level-tol, level+tol] but closes < level-tol.
func countLevelTests(level float64, sType SwingType, klines []market.KlineBar, startIndex int, tolerancePct float64) int {
	if startIndex >= len(klines) {
		return 0
	}

	testedCount := 0
	upperBound := level * (1 + tolerancePct)
	lowerBound := level * (1 - tolerancePct)

	inTestZone := false

	for i := startIndex; i < len(klines); i++ {
		c := klines[i]

		if sType == SwingLow { // Support Logic
			// 1. Check if price entered the tolerance zone (Low is within range)
			// User logic: if c.Low >= lowerBound && c.Low <= upperBound
			// Expanded to allow dipping slightly below but not closing below?
			// User logic: "If Low < lowerBound -> broken". So strictly within bounds for a "test".
			
			// Entering the zone
			if c.Low >= lowerBound && c.Low <= upperBound {
				inTestZone = true
			}

			// Valid bounce (Test Confirmed)
			// User logic: if inTestZone && c.Close > upperBound
			if inTestZone && c.Close > upperBound {
				testedCount++
				inTestZone = false // Reset for next test
			}

			// Broken Support
			// User logic: if c.Low < lowerBound
			if c.Low < lowerBound {
				inTestZone = false // Support broken, stop counting this sequence?
				// Usually if support is broken, it's no longer a support.
				// But we just stop the current "test" state. Future tests might be retests from below (resistance)?
				// For simplicity, we just reset.
			}

		} else { // SwingHigh (Resistance Logic) - Symmetric
			// Entering the zone (High is within range)
			if c.High >= lowerBound && c.High <= upperBound {
				inTestZone = true
			}

			// Valid rejection (Test Confirmed)
			// For resistance, we want Close < lowerBound (rejected down)
			if inTestZone && c.Close < lowerBound {
				testedCount++
				inTestZone = false
			}

			// Broken Resistance
			// If High > upperBound
			if c.High > upperBound {
				inTestZone = false
			}
		}
	}
	return testedCount
}

func analyzeTrend(swings []SwingPoint) string {
	if len(swings) < 3 {
		return "Insufficient data"
	}
	
	// Analyze the sequence of labels
	// We want to see consecutive HH+HL or LH+LL
	
	// Get last few swings
	startIdx := 0
	if len(swings) > 6 {
		startIdx = len(swings) - 6
	}
	recent := swings[startIdx:]
	
	hh := 0
	hl := 0
	lh := 0
	ll := 0
	
	for _, s := range recent {
		switch s.Label {
		case "HH": hh++
		case "HL": hl++
		case "LH": lh++
		case "LL": ll++
		}
	}
	
	if hh >= 2 && hl >= 1 {
		return "Strong Uptrend (Consecutive HHs)"
	}
	if lh >= 2 && ll >= 1 {
		return "Strong Downtrend (Consecutive LHs)"
	}
	if hh >= 1 && hl >= 1 {
		return "Uptrend Structure (HH+HL)"
	}
	if lh >= 1 && ll >= 1 {
		return "Downtrend Structure (LH+LL)"
	}
	
	// Early signal detection for 3 points (e.g. H -> L -> LH)
	if len(swings) == 3 {
		last := swings[len(swings)-1]
		if last.Label == "LH" {
			return "Potential Downtrend (Lower High observed)"
		}
		if last.Label == "LL" {
			return "Potential Downtrend (Lower Low observed)"
		}
		if last.Label == "HH" {
			return "Potential Uptrend (Higher High observed)"
		}
		if last.Label == "HL" {
			return "Potential Uptrend (Higher Low observed)"
		}
	}
	
	return "Mixed/Consolidation"
}

// FormatFeaturesToText generates the user-requested string block
func (f *TechnicalFeatures) FormatFeaturesToText(maxItems int) string {
	if len(f.SwingPoints) == 0 {
		return "### Physical Structural Anchors:\n(No swing points identified in recent data)\n\n"
	}

	var sb strings.Builder

	// Filter to show only the last N items
	count := len(f.SwingPoints)
	start := 0
	if maxItems > 0 && count > maxItems {
		start = count - maxItems
	}

	// Title
	sb.WriteString(fmt.Sprintf("### Physical Structural Anchors (Recent %d Swings):\n", count-start))

	for i := start; i < count; i++ {
		s := f.SwingPoints[i]
		
		// Label formatting: [Higher High (HH)] or [Swing High]
		labelStr := ""
		if s.Label != "" && s.Label != "High" && s.Label != "Low" {
			// Map short codes to full names
			fullLabel := ""
			switch s.Label {
			case "HH": fullLabel = "Higher High"
			case "HL": fullLabel = "Higher Low"
			case "LH": fullLabel = "Lower High"
			case "LL": fullLabel = "Lower Low"
			case "EH": fullLabel = "Equal High"
			case "EL": fullLabel = "Equal Low"
			default: fullLabel = s.Label
			}
			labelStr = fmt.Sprintf("[%s (%s)]", fullLabel, s.Label)
		} else {
			// Fallback for first points
			if s.Type == SwingHigh {
				labelStr = "[Swing High]"
			} else {
				labelStr = "[Swing Low]"
			}
		}

		timeStr := time.Unix(s.Time/1000, 0).UTC().Format("01-02 15:04")
		
		// Format line
		// - [Higher Low (HL)]: 67763.70 | Tested: 3 times (02-22 00:00)
		sb.WriteString(fmt.Sprintf("- %-20s: %-9s | Tested: %d times (%s)\n",
			labelStr,
			formatPriceForPrompt(s.Price),
			s.TestCount,
			timeStr,
		))
	}
	
	if f.Trend != "" {
		sb.WriteString(fmt.Sprintf("(Trend Evidence: %s)\n", f.Trend))
	}
	sb.WriteString("\n")

	return sb.String()
}
