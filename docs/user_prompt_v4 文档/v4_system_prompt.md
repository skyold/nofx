# Chaos Trading System - System Prompt V4.0
## Structural Facts Edition (结构事实版本)

---

## 🎯 设计哲学

**核心原则**：Signal只负责"计算外包"，不负责"方向判断"

```
V2: LLM作为规则匹配器 (Rule Matcher)
V3: LLM作为分数优化器 (Score Optimizer)
V4: LLM作为结构推理器 (Structural Reasoner)
```

**关键特性**：
- ❌ **禁止方向词**（bullish/bearish/long/short）
- ❌ **禁止综合评分**（conviction_score/trend_score）
- ✅ **只提供结构事实**（higher_high_count/rsi_value）
- ✅ **完全可验证**（每个值都可复算）
- ✅ **判断权在LLM**（不预判方向）

---

## 📐 Signal设计三大原则

### 原则 1：不可出现方向词

**禁止**：
- bullish / bearish
- long / short
- uptrend / downtrend
- breakout / reversal（已带解释）
- rising / falling（已带方向）
- strong / weak（已带判断）
- confirmation / divergence（已带解释）

**允许**：
- higher_high（客观事实）
- lower_low（客观事实）
- range_high_touch（位置关系）
- volatility_percentile（统计位置）
- ema20_above_ema50（位置关系）
- structure_break_high（事实检测）

---

### 原则 2：必须可验证、可复算

每一个signal必须：
1. **有明确计算公式**
2. **无主观参数**
3. **不依赖解释**
4. **可以用代码验证**

**示例**：

✅ **正确**：
```python
higher_high_count_5 = count(high[i] > high[i-1] for i in range(-5, 0))
# 明确：数最近5根有多少次创新高
```

❌ **错误**：
```python
trend_strength = calculate_subjective_strength(data)
# 不明确：什么是"strength"？如何计算？
```

---

### 原则 3：保持"结构粒度"

Signal只提供：
- **状态**（true/false）
- **次数**（count）
- **强度数值**（原始值）
- **百分位**（统计位置）

**禁止提供"总结评分"**：

❌ **错误**：
```json
"trend_score": 0.82,              // 压缩判断
"market_bias": 0.6,               // 方向判断
"conviction_score": 0.35          // 综合评分
```

✅ **正确**：
```json
"higher_high_count_5": 3,         // 原子事实
"rsi_value": 68,                  // 原始值
"rsi_percentile_200": 85,         // 统计位置
"ema20_above_ema50": true         // 关系事实
```

---

## 🏗️ Signal 6层分层结构

```
signals:
  structure:      # 结构层
  momentum:       # 动量层
  volatility:     # 波动层
  liquidity:      # 流动性层
  positioning:    # 持仓结构层（可选）
  ranking:        # 横向排名层（可选）
```

---

## 1️⃣ Structure Layer (结构层)

**目标**：告诉LLM当前价格结构状态

### 1.1 Higher High / Lower Low 统计

```python
higher_high_count_N
lower_low_count_N
```

**计算方法**：
```python
# 在最近N根K线内
higher_high_count = 0
for i in range(len(klines)-N, len(klines)):
    if klines[i].high > klines[i-1].high:
        higher_high_count += 1

lower_low_count = 0
for i in range(len(klines)-N, len(klines)):
    if klines[i].low < klines[i-1].low:
        lower_low_count += 1
```

**配置参数**：
- N = 5（短期）
- N = 10（中期）
- N = 20（长期）

**示例输出**：
```json
"higher_high_count_5": 3,
"higher_high_count_10": 5,
"lower_low_count_5": 1,
"lower_low_count_10": 3
```

**LLM理解**：
```
HH_5=3, LL_5=1 → 最近5根更多创新高
HH_10=5, LL_10=3 → 中期也偏向创高
→ LLM推理：可能是上升结构
```

---

### 1.2 结构突破检测

```python
structure_break_high
structure_break_low
```

**计算方法**：
```python
# 使用最近N根K线作为参考范围（推荐N=20）
lookback_high = max(high[-N:])
lookback_low = min(low[-N:])

structure_break_high = current_high > lookback_high
structure_break_low = current_low < lookback_low
```

**说明**：
- 这**不是breakout信号**（breakout是方向词）
- 只是客观检测：是否突破过去N根的结构范围
- 不判断突破的有效性

**示例输出**：
```json
"structure_break_high": true,
"structure_break_low": false
```

**LLM理解**：
```
break_high=true → 突破了20根高点
break_low=false → 没有突破20根低点
→ LLM推理：可能在测试上方阻力
```

---

### 1.3 区间压缩度

```python
range_compression_ratio
```

**计算方法**：
```python
# 最近N根的高低点范围，除以ATR归一化
N = 20  # 推荐值

range_high = max(high[-N:])
range_low = min(low[-N:])
range_width = range_high - range_low

atr_n = calculate_atr(N)

range_compression_ratio = range_width / atr_n
```

**数值含义**：
- `< 2.0`: 极度压缩（突破前夕）
- `2.0 - 4.0`: 正常范围
- `> 4.0`: 宽幅震荡

**示例输出**：
```json
"range_compression_ratio": 1.85
```

**LLM理解**：
```
ratio=1.85 < 2.0 → 区间压缩
→ LLM推理：可能即将突破（但不知道方向）
```

---

### 1.4 Swing高低点计数

```python
swing_high_count_N
swing_low_count_N
```

**计算方法**（Swing定义）：
```python
def is_swing_high(i, lookback=2):
    """
    Swing High: 左右各lookback根K线都低于当前high
    """
    if i < lookback or i >= len(klines) - lookback:
        return False
    
    current_high = klines[i].high
    
    # 检查左边lookback根
    for j in range(i-lookback, i):
        if klines[j].high >= current_high:
            return False
    
    # 检查右边lookback根
    for j in range(i+1, i+lookback+1):
        if klines[j].high >= current_high:
            return False
    
    return True

# 统计最近N根内的Swing High数量
swing_high_count = sum(is_swing_high(i) for i in range(len(klines)-N, len(klines)))
```

**示例输出**：
```json
"swing_high_count_20": 3,
"swing_low_count_20": 2
```

**LLM理解**：
```
swing_high=3, swing_low=2 → 更多明显的高点
→ LLM推理：结构可能在形成阻力位
```

---

## 2️⃣ Momentum Layer (动量层)

**目标**：提供原始动量指标，不给方向判断

### 2.1 RSI 状态

```python
rsi_value
rsi_percentile_200
rsi_over_70
rsi_below_30
```

**计算方法**：
```python
# 标准RSI(14)
rsi_value = calculate_rsi(close, period=14)

# 计算RSI在过去200根中的百分位
rsi_history = [calculate_rsi(close[:i], 14) for i in range(len(close)-200, len(close))]
rsi_percentile_200 = percentile_rank(rsi_value, rsi_history)

# 客观事实检测
rsi_over_70 = (rsi_value > 70)
rsi_below_30 = (rsi_value < 30)
```

**示例输出**：
```json
"rsi_value": 68.5,
"rsi_percentile_200": 85,
"rsi_over_70": false,
"rsi_below_30": false
```

**LLM理解**：
```
RSI=68.5, 在85%分位 → 动量较强但未超买
over_70=false → 还没到传统超买区
→ LLM推理：有一定动量但还有空间
```

---

### 2.2 EMA 关系

```python
ema20_above_ema50
ema20_slope
ema50_slope
ema_distance_percent
```

**计算方法**：
```python
ema20 = calculate_ema(close, 20)
ema50 = calculate_ema(close, 50)

# 位置关系（客观事实）
ema20_above_ema50 = (ema20[-1] > ema50[-1])

# 斜率（不是"上升"/"下降"，只是数值）
ema20_slope = ema20[-1] - ema20[-2]
ema50_slope = ema50[-1] - ema50[-2]

# 距离（归一化）
ema_distance_percent = ((ema20[-1] - ema50[-1]) / close[-1]) * 100
```

**示例输出**：
```json
"ema20_above_ema50": true,
"ema20_slope": 42.3,
"ema50_slope": 18.7,
"ema_distance_percent": 1.8
```

**LLM理解**：
```
ema20 > ema50 → 短期在长期上方
slope_20=42.3 > slope_50=18.7 → 短期斜率更陡
distance=1.8% → 有一定分离
→ LLM推理：短期可能比长期强
```

**注意**：不说"uptrend"，只给斜率数值！

---

### 2.3 MACD 状态

```python
macd_line_value
macd_signal_value
macd_histogram_value
macd_histogram_positive
```

**计算方法**：
```python
macd_line, macd_signal, macd_histogram = calculate_macd(close, 12, 26, 9)

macd_line_value = macd_line[-1]
macd_signal_value = macd_signal[-1]
macd_histogram_value = macd_histogram[-1]

# 客观事实
macd_histogram_positive = (macd_histogram_value > 0)
```

**示例输出**：
```json
"macd_line_value": 38.5,
"macd_signal_value": 32.1,
"macd_histogram_value": 6.4,
"macd_histogram_positive": true
```

**LLM理解**：
```
histogram=6.4, positive=true → MACD在信号线上方
histogram在增加中 → 动量可能在增强
→ LLM推理：短期动量可能偏强
```

---

### 2.4 连续K线方向统计

```python
consecutive_up_bars
consecutive_down_bars
up_bars_count_N
down_bars_count_N
```

**计算方法**：
```python
# 连续上涨/下跌K线数
consecutive_up_bars = 0
for i in range(len(klines)-1, 0, -1):
    if klines[i].close > klines[i].open:
        consecutive_up_bars += 1
    else:
        break

consecutive_down_bars = 0
for i in range(len(klines)-1, 0, -1):
    if klines[i].close < klines[i].open:
        consecutive_down_bars += 1
    else:
        break

# 最近N根内的上涨/下跌K线数
N = 10
up_bars_count = sum(1 for i in range(-N, 0) if klines[i].close > klines[i].open)
down_bars_count = sum(1 for i in range(-N, 0) if klines[i].close < klines[i].open)
```

**示例输出**：
```json
"consecutive_up_bars": 3,
"consecutive_down_bars": 0,
"up_bars_count_10": 7,
"down_bars_count_10": 3
```

**LLM理解**：
```
连续3根阳线
最近10根中7根阳线
→ LLM推理：最近可能偏强
```

---

## 3️⃣ Volatility Layer (波动层)

**目标**：提供波动率统计，不判断"高"或"低"

### 3.1 ATR 百分位

```python
atr_value
atr_percentile_200
```

**计算方法**：
```python
atr_value = calculate_atr(close, high, low, period=14)

# 计算在过去200根中的百分位
atr_history = [calculate_atr(close[:i], high[:i], low[:i], 14) 
               for i in range(len(close)-200, len(close))]
atr_percentile_200 = percentile_rank(atr_value, atr_history)
```

**示例输出**：
```json
"atr_value": 312.5,
"atr_percentile_200": 78
```

**LLM理解**：
```
ATR在78%分位 → 波动率高于过去78%的时间
→ LLM推理：当前波动较大（但不说"high volatility"）
```

---

### 3.2 实体比例

```python
body_ratio
```

**计算方法**：
```python
body = abs(close - open)
full_range = high - low

if full_range == 0:
    body_ratio = 0
else:
    body_ratio = body / full_range
```

**数值含义**：
- `> 0.7`: 实体占比大（单边推动）
- `0.3 - 0.7`: 正常
- `< 0.3`: 实体占比小（上下影线长）

**示例输出**：
```json
"body_ratio": 0.64
```

**LLM理解**：
```
body_ratio=0.64 → 实体占比64%
→ LLM推理：有一定单边性（但不判断方向）
```

---

### 3.3 Bollinger Bands宽度

```python
bb_width_percent
bb_width_percentile_200
```

**计算方法**：
```python
bb_upper, bb_middle, bb_lower = calculate_bollinger_bands(close, 20, 2)

# 宽度百分比
bb_width = bb_upper - bb_lower
bb_width_percent = (bb_width / bb_middle) * 100

# 宽度百分位
bb_width_history = [calculate_bb_width(close[:i], 20, 2) 
                    for i in range(len(close)-200, len(close))]
bb_width_percentile_200 = percentile_rank(bb_width_percent, bb_width_history)
```

**示例输出**：
```json
"bb_width_percent": 4.8,
"bb_width_percentile_200": 25
```

**LLM理解**：
```
BB宽度在25%分位 → 比过去75%的时间更窄
→ LLM推理：波动收缩（可能突破前夕）
```

---

### 3.4 价格在Bollinger Bands的位置

```python
price_bb_position
```

**计算方法**：
```python
# 价格在BB中的位置（0-100）
bb_upper, bb_middle, bb_lower = calculate_bollinger_bands(close, 20, 2)

if bb_upper == bb_lower:
    price_bb_position = 50
else:
    price_bb_position = ((close[-1] - bb_lower) / (bb_upper - bb_lower)) * 100
```

**示例输出**：
```json
"price_bb_position": 88.5
```

**LLM理解**：
```
position=88.5 → 价格在BB的88.5%位置
→ LLM推理：接近上轨（但不说"overbought"）
```

---

## 4️⃣ Liquidity Layer (流动性层)

**目标**：检测流动性扫单和成交量异常

### 4.1 流动性扫单检测

```python
liquidity_sweep_high
liquidity_sweep_low
```

**计算方法**：
```python
N = 10  # 回看周期

# 扫高点逻辑
lookback_high = max(high[-N-1:-1])  # 排除当前K线
current_high = high[-1]
current_close = close[-1]

# 条件：突破了过去高点，但收盘回落
liquidity_sweep_high = (current_high > lookback_high) and (current_close < lookback_high)

# 扫低点逻辑
lookback_low = min(low[-N-1:-1])
current_low = low[-1]

# 条件：跌破了过去低点，但收盘回升
liquidity_sweep_low = (current_low < lookback_low) and (current_close > lookback_low)
```

**说明**：
- 这是客观的价格行为检测
- 不判断是"假突破"还是"真突破"
- LLM自己推理含义

**示例输出**：
```json
"liquidity_sweep_high": false,
"liquidity_sweep_low": true
```

**LLM理解**：
```
sweep_low=true → 价格跌破了低点但又收回
→ LLM推理：可能是止损猎杀
```

---

### 4.2 成交量统计

```python
volume_value
volume_percentile_200
volume_spike
volume_ma_ratio
```

**计算方法**：
```python
volume_value = volume[-1]

# 成交量百分位
volume_history = volume[-200:]
volume_percentile_200 = percentile_rank(volume_value, volume_history)

# 成交量MA比率
volume_ma = mean(volume[-20:])
volume_ma_ratio = volume_value / volume_ma

# 成交量突增检测
volume_spike = (volume_ma_ratio > 1.5)
```

**示例输出**：
```json
"volume_value": 1523400,
"volume_percentile_200": 82,
"volume_spike": true,
"volume_ma_ratio": 1.68
```

**LLM理解**：
```
volume在82%分位，spike=true
volume是MA的1.68倍
→ LLM推理：成交量明显放大
```

---

### 4.3 价格与成交量的同步性

```python
price_volume_sync
```

**计算方法**：
```python
# 价格变化方向
price_change = close[-1] - close[-2]
price_up = (price_change > 0)

# 成交量变化方向
volume_change = volume[-1] - volume[-2]
volume_up = (volume_change > 0)

# 是否同步（都增加或都减少）
price_volume_sync = (price_up == volume_up)
```

**示例输出**：
```json
"price_volume_sync": true
```

**LLM理解**：
```
sync=true → 价格和成交量方向一致
（配合其他信号判断是涨还是跌）
→ LLM推理：量价配合
```

---

## 5️⃣ Positioning Layer (持仓结构层) - 可选

**前提**：需要有多空比、持仓量、资金费率数据

### 5.1 多空持仓比

```python
long_short_ratio
long_account_percent
short_account_percent
```

**计算方法**：
```python
# 从交易所API获取
long_positions = get_long_positions(symbol)
short_positions = get_short_positions(symbol)

long_short_ratio = long_positions / short_positions if short_positions > 0 else 0

total = long_positions + short_positions
long_account_percent = (long_positions / total) * 100 if total > 0 else 0
short_account_percent = (short_positions / total) * 100 if total > 0 else 0
```

**示例输出**：
```json
"long_short_ratio": 1.85,
"long_account_percent": 64.9,
"short_account_percent": 35.1
```

**LLM理解**：
```
多空比1.85，多头占65%
→ LLM推理：多头占优（但不判断是好是坏）
```

---

### 5.2 持仓量变化

```python
oi_value
oi_change_percent
oi_change_percentile_100
```

**计算方法**：
```python
oi_value = get_open_interest(symbol)
oi_prev = get_open_interest_history(symbol, periods=1)[0]

oi_change_percent = ((oi_value - oi_prev) / oi_prev) * 100

# 持仓量变化的百分位
oi_change_history = get_oi_change_history(symbol, periods=100)
oi_change_percentile_100 = percentile_rank(oi_change_percent, oi_change_history)
```

**示例输出**：
```json
"oi_value": 1234567890,
"oi_change_percent": 3.4,
"oi_change_percentile_100": 78
```

**LLM理解**：
```
OI增加3.4%，在78%分位
→ LLM推理：持仓量增长较快
```

---

### 5.3 资金费率

```python
funding_rate
funding_rate_percentile_100
```

**计算方法**：
```python
funding_rate = get_funding_rate(symbol)

# 资金费率百分位
funding_history = get_funding_rate_history(symbol, periods=100)
funding_rate_percentile_100 = percentile_rank(funding_rate, funding_history)
```

**示例输出**：
```json
"funding_rate": 0.0125,
"funding_rate_percentile_100": 85
```

**LLM理解**：
```
funding_rate=0.0125%，在85%分位
→ LLM推理：资金费率较高（多头需要支付）
```

---

## 6️⃣ Ranking Layer (横向排名层) - 可选

**目标**：提供在全市场中的相对位置

### 6.1 成交量排名

```python
volume_rank_24h
volume_rank_total_symbols
```

**计算方法**：
```python
# 获取24小时成交量排名
all_symbols_volume = get_all_symbols_volume_24h()
current_symbol_volume = volume_24h(current_symbol)

# 排名（1 = 最高）
volume_rank_24h = sorted(all_symbols_volume.values(), reverse=True).index(current_symbol_volume) + 1

volume_rank_total_symbols = len(all_symbols_volume)
```

**示例输出**：
```json
"volume_rank_24h": 12,
"volume_rank_total_symbols": 350
```

**LLM理解**：
```
24小时成交量排名12/350
→ LLM推理：交易活跃度较高
```

---

### 6.2 波动率排名

```python
volatility_rank_24h
volatility_rank_total_symbols
```

**计算方法**：
```python
# 获取24小时波动率排名（用ATR或价格变化幅度）
all_symbols_volatility = get_all_symbols_volatility_24h()
current_symbol_volatility = calculate_volatility_24h(current_symbol)

volatility_rank_24h = sorted(all_symbols_volatility.values(), reverse=True).index(current_symbol_volatility) + 1

volatility_rank_total_symbols = len(all_symbols_volatility)
```

**示例输出**：
```json
"volatility_rank_24h": 5,
"volatility_rank_total_symbols": 350
```

**LLM理解**：
```
波动率排名5/350
→ LLM推理：波动率较高的币种
```

---

### 6.3 相对强度排名

```python
relative_strength_rank_24h
price_change_percent_24h
```

**计算方法**：
```python
# 24小时价格变化
price_24h_ago = get_price_24h_ago(current_symbol)
price_current = close[-1]

price_change_percent_24h = ((price_current - price_24h_ago) / price_24h_ago) * 100

# 获取排名
all_symbols_change = get_all_symbols_price_change_24h()
relative_strength_rank_24h = sorted(all_symbols_change.values(), reverse=True).index(price_change_percent_24h) + 1
```

**示例输出**：
```json
"relative_strength_rank_24h": 28,
"price_change_percent_24h": 2.85
```

**LLM理解**：
```
24小时涨幅2.85%，排名28/350
→ LLM推理：表现中等偏上
```

---

## ✅ 总结

### V4的核心价值

1. **信息无损** - 不压缩成评分
2. **判断权在LLM** - 不预判方向
3. **完全可验证** - 每个值都可复算
4. **系统工程级** - 分层清晰

### V4 vs V2/V3

```
V2: 给LLM标签 → LLM匹配规则
V3: 给LLM分数 → LLM组合分数
V4: 给LLM事实 → LLM结构推理
```

**V4是真正的"结构推理时代"！**
