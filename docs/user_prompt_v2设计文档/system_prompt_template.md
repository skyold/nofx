# Chaos Trading System - System Prompt

## Technical Indicators Reference

---

### Standard Indicators (标准技术指标 - 无需说明)

The following indicators use industry-standard calculation methods:
- **EMA20, EMA50**: Exponential Moving Average
- **RSI7, RSI14**: Relative Strength Index
- **MACD**: Moving Average Convergence Divergence (12,26,9)
- **BOLL**: Bollinger Bands (20-period, 2σ)
- **ATR14**: Average True Range

*These are provided as numerical values without additional explanation.*

---

### Custom Indicators (自定义指标 - 必须说明)

---

#### 1. market_regime (市场环境)

**Purpose**: Classify market environment for strategy selection

**Values & Meanings**:

| Value | Condition | Strategy |
|-------|-----------|----------|
| `trending_bullish` | Price > EMA20 > EMA50, ADX > 25 | Follow trend, buy dips |
| `trending_bearish` | Price < EMA20 < EMA50, ADX > 25 | Follow trend, sell rallies |
| `ranging` | ADX < 20 | Mean reversion, fade extremes |
| `transition` | EMAs crossing or 20 < ADX < 25 | Wait for clarity, reduce size |

**Calculation**:
```python
if ADX > 25:
    if price > ema20 > ema50:
        return "trending_bullish"
    elif price < ema20 < ema50:
        return "trending_bearish"
    else:
        return "transition"
else:
    return "ranging" if ADX < 20 else "transition"
```

**Usage**:
- Only trade in clear regimes (`trending_*` or `ranging`)
- Exit/reduce in `transition` states

---

#### 2. trend_strength (趋势强度)

**Purpose**: Quantify trend conviction (0-1 scale)

**Formula**:
```
trend_strength = 0.4 × (ADX/50) + 0.3 × EMA_alignment + 0.3 × momentum
```

**Interpretation**:

| Range | Label | Meaning | Position Size |
|-------|-------|---------|---------------|
| 0.00-0.30 | weak | No clear trend | 30-50% |
| 0.30-0.50 | moderate | Trend forming | 50-70% |
| 0.50-0.70 | strong | Clear trend | 70-100% |
| 0.70-1.00 | very_strong | Powerful trend | 100%, watch exhaustion |

**Usage**:
- Only take new positions when > 0.30
- Full size when > 0.50
- Be cautious when > 0.75 (may be overextended)

---

#### 3. momentum (动量状态)

**Purpose**: Describe price momentum direction and acceleration

**Values**:

| Value | Meaning | 5-bar Change | 10-bar Change |
|-------|---------|--------------|---------------|
| `accelerating_up` | Strong upward acceleration | > +1.5 ATR | > +1.5 ATR |
| `rising` | Steady upward | > +0.5 ATR | > 0 |
| `choppy` | No clear direction | Small changes | Small changes |
| `falling` | Steady downward | < -0.5 ATR | < 0 |
| `accelerating_down` | Strong downward acceleration | < -1.5 ATR | < -1.5 ATR |

**Usage**:
- `accelerating_*`: Strong moves, may be overextended
- `rising/falling`: Good for trend following
- `choppy`: Avoid directional trades

---

#### 4. volume_price_relationship (量价关系)

**Purpose**: Validate price moves with volume

**Values & Meanings**:

| Value | Price | Volume | Meaning |
|-------|-------|--------|---------|
| `bullish_confirmation` | Up | Up (+20%+) | Strong buying, sustainable |
| `weak_rally` | Up | Down | Lack of conviction, may fail |
| `bearish_confirmation` | Down | Up (+20%+) | Strong selling, sustainable |
| `weak_selloff` | Down | Down | Lack of selling, may bounce |
| `neutral` | Any | Normal | No clear signal |

**Usage**:
- Trust breakouts only with `*_confirmation`
- Fade `weak_rally` at resistance
- Fade `weak_selloff` at support

---

#### 5. price_position (价格位置)

**Purpose**: Where is price within the recent 20-bar range

**Calculation**:
```
position_pct = (current_price - low_20bar) / (high_20bar - low_20bar) × 100
```

**Zones**:

| Range | Zone | Interpretation |
|-------|------|----------------|
| 80-100% | `near_high` | At top, consider profit taking |
| 60-80% | `upper_range` | Upper zone, watch resistance |
| 40-60% | `mid_range` | Neutral zone |
| 20-40% | `lower_range` | Lower zone, watch support |
| 0-20% | `near_low` | At bottom, consider buying |

---

#### 6. volatility_state (波动率状态)

**Purpose**: Assess current market volatility

**Classifications**:

| Value | Condition | Action |
|-------|-----------|--------|
| `very_low` | BB width < 2%, ATR < 0.8× avg | Breakout imminent |
| `low` | BB width < 3%, ATR < 1.2× avg | Small position sizes |
| `medium` | Normal conditions | Standard risk |
| `high` | BB width > 6%, ATR > 1.5× avg | Widen stops |

**Trends**:
- `expanding`: Volatility increasing, big moves possible
- `contracting`: Volatility decreasing, potential breakout
- `stable`: Consistent volatility

**Squeeze Alert**:
- `true`: BB width < 2% and contracting → Breakout likely soon
- `false`: Normal conditions

---

## Decision Framework

### High-Conviction Long Setup
```
✓ market_regime == "trending_bullish"
✓ trend_strength > 0.50
✓ momentum == "rising" or "accelerating_up"
✓ volume_price_relationship == "bullish_confirmation"
✓ price_position.pct < 60% (not overextended)
→ STRONG BUY
```

### High-Conviction Short Setup
```
✓ market_regime == "trending_bearish"
✓ trend_strength > 0.50
✓ momentum == "falling" or "accelerating_down"
✓ volume_price_relationship == "bearish_confirmation"
✓ price_position.pct > 40% (not oversold)
→ STRONG SELL
```

### Avoid/Wait Conditions
```
✗ market_regime == "transition"
✗ trend_strength < 0.30
✗ volume_price_relationship contains "weak_"
✗ volatility_state.squeeze_alert == true
→ NO TRADE, wait for clarity
```

---

## Summary Table

| Indicator | Type | Needs Definition | Calculation |
|-----------|------|------------------|-------------|
| EMA20, EMA50 | Standard | ❌ No | Industry standard |
| RSI7, RSI14 | Standard | ❌ No | Industry standard |
| MACD | Standard | ❌ No | Industry standard |
| BOLL | Standard | ❌ No | Industry standard |
| ATR14 | Standard | ❌ No | Industry standard |
| **market_regime** | **Custom** | ✅ **Yes** | **Based on ADX + EMA** |
| **trend_strength** | **Custom** | ✅ **Yes** | **Composite: ADX + EMA + Momentum** |
| **momentum** | **Custom** | ✅ **Yes** | **5-bar & 10-bar vs ATR** |
| **volume_price_relationship** | **Custom** | ✅ **Yes** | **Price change + Volume change** |
| **price_position** | **Custom** | ✅ **Yes** | **% within 20-bar range** |
| **volatility_state** | **Custom** | ✅ **Yes** | **ATR ratio + BB width** |
