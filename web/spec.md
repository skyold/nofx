# NoFX AI Agent Trading Platform - Brand & UI Design Specification

## 品牌核心定位

**产品定位**: AI-Native Multi-Asset Trading Platform

**核心卖点**:
- 🤖 AI Agent 自动交易
- 🌍 多资产统一账户
- 📊 专业级资产管理

**品牌气质参考**:
- Stripe（极简金融科技）
- Robinhood（年轻投资者）
- OpenAI（AI 科技感）
- TradingView（专业交易界面）

**视觉关键词**:
- AI Trading Terminal
- Minimal Fintech
- Dark SaaS
- Future Finance

---

## 产品功能介绍

### 核心功能

#### 1️⃣ 全品类资产聚合

**功能描述**: 打通股票、外汇、加密货币、大宗商品等主流交易平台 API，实现一个账户管理全球多类别资产，自由切换各类资产交易。

**支持资产**:
- 📈 **股票**: 美股、港股、A 股等全球主要股市
- 💱 **外汇**: 主要货币对、交叉货币对
- 🪙 **加密货币**: BTC、ETH 等主流数字货币
- 🛢️ **大宗商品**: 黄金、原油、农产品等

**UI 展示重点**:
- 多资产统一 Dashboard
- 资产类别切换器
- 跨市场持仓汇总
- 统一盈亏分析

---

#### 2️⃣ AI 智能交易策略

**功能描述**: 用户可以开发提示词交给 AI 进行自动交易，或者进行人在回路 AI 辅助的交易。是个 AI agent 驱动的交易平台。

**交易模式**:
- 🤖 **全自动 AI 交易**: 用户配置提示词，AI Agent 自动执行交易策略
- 👤 **人在回路 AI 辅助**: AI 提供信号和建议，用户确认执行
- 📝 **提示词工作室**: 可视化提示词编辑、测试、优化
- 🧠 **持续学习**: AI 模型根据市场反馈持续优化

**UI 展示重点**:
- AI Agent 状态面板
- 提示词编辑器
- 策略回测结果
- AI 信号展示
- 人机交互确认界面

---

#### 3️⃣ 持续扩展的生态系统

**功能描述**: 通过持续拓展可接入资产类别，迭代 AI 交易模型，让普通投资者也能轻松实现跨市场、多品类的专业级资产配置与交易，成为一站式智能资产管理的标杆平台。

**扩展方向**:
- 🔌 **新资产接入**: 持续集成更多交易市场和资产类别
- 🔄 **AI 模型迭代**: 引入更先进的 LLM 和量化模型
- 🎯 **专业工具**: 高级图表、技术指标、风险管理工具
- 🌐 **社区生态**: 策略分享、信号市场、社交交易

**UI 展示重点**:
- 可扩展的模块化设计
- 插件/集成管理界面
- 版本更新提示
- 社区功能入口

---

### 软件架构特点

#### 技术优势
- **AI Agent 驱动**: 基于先进 LLM 的智能交易决策
- **多交易所支持**: 统一 API 抽象层，支持主流交易平台
- **实时数据处理**: 低延迟市场数据 feed
- **安全合规**: API Key 加密存储，权限隔离
- **高可用性**: 分布式架构，99.99% 可用性

#### 用户体验
- **一站式管理**: 单界面管理全球多资产
- **智能辅助**: AI 提供专业级交易建议
- **低门槛**: 普通投资者也能使用专业工具
- **透明可控**: 所有 AI 决策可解释、可干预

---

## 功能模块 UI 展示指南

### 1. 全品类资产聚合 - UI 设计

#### Dashboard 主界面
```
-------------------------------------------------
|  资产总览 (Total Balance)                    |
|  $125,847.32  (+12.5% 24h)                   |
|                                              |
|  [股票] [外汇] [Crypto] [商品]  ← Tab 切换   |
-------------------------------------------------
|  持仓分布 (Pie Chart)  |  24h 盈亏趋势        |
|  - Crypto: 45%         |  [Chart]            |
|  - Stocks: 30%         |                     |
|  - Forex: 15%          |                     |
|  - Commodities: 10%    |                     |
-------------------------------------------------
|  热门资产实时价格 Ticker                    |
|  BTC $67,234  ETH $3,456  AAPL $178.32      |
-------------------------------------------------
```

**设计要点**:
- 使用品牌主色 `#5B5CFF` 突出总资产
- 盈亏使用交易颜色：绿涨 `#00C853` / 红跌 `#FF3B3B`
- Tab 切换使用 AI 高亮色 `#00D4FF` 标记当前选中
- 图表使用渐变填充，增强科技感

---

### 2. AI 智能交易策略 - UI 设计

#### AI Agent 状态面板
```
┌─────────────────────────────────────────────┐
│  🤖 AI Agent Status         [●] Active     │
├─────────────────────────────────────────────┤
│  Strategy: Trend Following v2.1             │
│  Market Regime: TRENDING (Bull)             │
│  Risk Level: LOW                            │
│  Confidence: 87%                            │
├─────────────────────────────────────────────┤
│  Current Positions:                         │
│  ✅ BTC Long  | +$1,234 (5.2%)             │
│  ⏸️  ETH Hold  | $0 (0.0%)                 │
│  🔍 SOL Scan  | Looking for entry          │
├─────────────────────────────────────────────┤
│  [停止 Agent] [调整参数] [查看日志]         │
└─────────────────────────────────────────────┘
```

**设计要点**:
- AI 状态使用 Cyber Blue `#00D4FF` 脉冲动画
- 置信度使用环形进度条展示
- 仓位状态使用图标 + 颜色直观显示
- 按钮使用品牌色，Hover 时发光

---

#### 提示词工作室（Prompt Studio）
```
┌─────────────────────────────────────────────┐
│  📝 Prompt Studio                           │
├─────────────────────────────────────────────┤
│  策略名称：Trend Following AI               │
│  ─────────────────────────────────────────  │
│  System Prompt:                             │
│  ┌─────────────────────────────────────┐   │
│  │ You are a professional trader...   │   │
│  │ Analyze market trends and...       │   │
│  │ [AI 自动补全提示]                   │   │
│  └─────────────────────────────────────┘   │
│                                              │
│  Data Format: [JSON ▼]                      │
│  Model: [Claude Opus 4.5 ▼]                 │
│  Risk Level: [○ Low ○ Medium ○ High]       │
│                                              │
│  [💾 保存] [🧪 回测] [🚀 部署]              │
└─────────────────────────────────────────────┘
```

**设计要点**:
- 编辑器使用深色背景 + JetBrains Mono 字体
- AI 补全提示使用高亮色下划线
- 回测按钮使用黄色（ Binance 风格）
- 部署按钮使用品牌主色 + 发光

---

### 3. 交易界面 - UI 设计

#### K 线图 + AI Insights
```
┌─────────────────────────────────────────────┐
│  BTC/USDT        $67,234  [+2.34%]         │
│  [15m] [1h] [4h] [1D] [1W]                  │
├─────────────────────────────────────────────┤
│                                             │
│         [K 线图区域 - TradingView 风格]      │
│         保持专业交易配色                     │
│                                             │
├─────────────────────────────────────────────┤
│  🧠 AI Insights                             │
│  ─────────────────────────────────────────  │
│  Trend Strength: ████████░░ Strong         │
│  Liquidity:      ██████░░░░ Increasing     │
│  Volatility:     ████░░░░░░ Low            │
│  AI Score:       8.5/10  [看涨]             │
│                                             │
│  Signal: BTC 可能在 $68,000 遇到阻力         │
│  Confidence: 78%                            │
└─────────────────────────────────────────────┘
```

**设计要点**:
- K 线图保持专业交易配色（绿涨红跌）
- AI Insights 面板使用品牌色边框
- 进度条使用渐变填充
- AI Score 使用环形仪表展示

---

#### 订单面板
```
┌─────────────────────────────────────────────┐
│  Place Order                                │
├─────────────────────────────────────────────┤
│  Type: [Market ▼]  Side: [Long / Short]    │
│                                              │
│  Amount (USDT): [___________]              │
│  Leverage:      [10x ◄────► 100x]          │
│                                              │
│  ─────────────────────────────────────────  │
│  Estimated Position:                        │
│  Size: $10,000  |  Leverage: 10x           │
│  Liq Price: $64,123  |  Margin: $1,000    │
│  ─────────────────────────────────────────  │
│                                              │
│  [买入/做多]  [卖出/做空]                   │
│   (Green)      (Red)                        │
└─────────────────────────────────────────────┘
```

**设计要点**:
- 买入按钮使用盈利绿色 `#00C853`
- 卖出按钮使用亏损红色 `#FF3B3B`
- 杠杆滑块使用品牌色进度条
- 预估数据使用等宽字体

---

### 4. 人在回路模式 - UI 设计

#### AI 信号确认界面
```
┌─────────────────────────────────────────────┐
│  🔔 AI Trading Signal                       │
├─────────────────────────────────────────────┤
│  Symbol: BTC/USDT                           │
│  Action: LONG                               │
│  Entry: $67,200 - $67,500                  │
│  Target: $69,000 (+2.7%)                   │
│  Stop Loss: $65,800 (-2.1%)                │
│  Leverage: 10x                              │
│                                              │
│  AI Reasoning:                              │
│  • 突破关键阻力位                            │
│  • 成交量放大 35%                           │
│  • RSI 显示超买但未极端                      │
│  • 资金费率中性偏多                          │
│                                              │
│  Confidence: 82%  |  Risk/Reward: 1:1.3    │
│                                              │
│  [❌ 拒绝]  [⏸️ 稍后]  [✅ 执行]            │
└─────────────────────────────────────────────┘
```

**设计要点**:
- 信号卡片使用品牌色边框 + 发光
- 置信度使用大字号突出显示
- 拒绝/执行按钮对比明显
- AI 理由使用列表清晰展示

---

### 5. 扩展生态 - UI 设计

#### 插件/集成管理
```
┌─────────────────────────────────────────────┐
│  🔌 Integrations & Extensions               │
├─────────────────────────────────────────────┤
│  Connected Exchanges:                       │
│  ✅ Binance    ✅ Bybit    ✅ Hyperliquid   │
│  ⭕ OKX       ⭕ Gate      ⭕ Kucoin         │
│                                              │
│  AI Models:                                 │
│  ✅ Claude Opus 4.5                         │
│  ✅ GPT-4 Turbo                             │
│  ⭕ Gemini Ultra                            │
│  ⭕ Llama 3 70B                             │
│                                              │
│  Coming Soon:                               │
│  📅 Interactive Brokers (Stocks)           │
│  📅 Forex.com (FX)                          │
│  📅 Gold/Silver APIs                        │
│                                              │
│  [+ Add Integration]                        │
└─────────────────────────────────────────────┘
```

**设计要点**:
- 已连接使用绿色勾选 + 品牌色背景
- 未连接使用灰色 + 虚线边框
- Coming Soon 使用渐变透明效果
- 添加按钮使用品牌主色

---

## 主题颜色系统（Design System）

### 主色（Brand Primary）

**Electric Indigo**: `#5B5CFF`

**用途**:
- Logo
- 按钮
- 主导航
- 关键操作

### AI 高亮色

**Cyber Blue**: `#00D4FF`

**用途**:
- AI Agent
- 提示
- Hover
- 数据高亮

### 背景色

- **Primary background**: `#0B0F1A`
- **Panel background**: `#121826`

### 交易颜色

- **盈利（Green）**: `#00C853`
- **亏损（Red）**: `#FF3B3B`

### 文本颜色

- **Primary text**: `#E6E8EF`
- **Secondary text**: `#8B93A7`

---

## 整体 UI 风格

### 核心特点

1. **深色交易终端**
   - 类似 TradingView 但更 AI
   - 长期使用舒适

2. **卡片式布局**
   - 模块：AI Agent, Market Scanner, Portfolio, Signals, Orders, Positions
   - 每个模块都是独立的 card panel

3. **微发光效果（AI 感）**
   - 按钮：`box-shadow: 0 0 12px rgba(0,212,255,0.35)`
   - AI 组件：glow border

---

## 字体系统

### 标题字体

**Inter** - 现代 SaaS 标准字体

### 数字字体

**JetBrains Mono** - 适合价格、PnL、数据

---

## Logo 设计方案

### 方案 1（推荐）- X Quantum

一个量子结构的 X，象征：
- AI
- 网络
- 交易连接

**颜色**: `#5B5CFF → #00D4FF` gradient

**风格**: 极简科技

### 方案 2 - AI Core

圆形核心，周围是 market nodes，象征 AI 控制市场

### 方案 3 - Signal Wave

交易信号波形形成字母 X

---

## 产品界面结构

### 主界面布局

```
------------------------------------------------ 
Top Bar 
AI Status | Account | Notifications 
------------------------------------------------ 

Left Navigation 
Markets 
Portfolio 
AI Agent 
Signals 
Strategies 
Orders 

------------------------------------------------ 
Main Workspace 

Market Scanner | Portfolio Summary 

Chart Area | AI Analysis 

Positions | Orders 
------------------------------------------------ 
```

---

## AI Agent 面板

AI Agent 是核心功能。

**界面内容**:
```
AI Agent Status

Strategy Running
Market Regime: TRENDING
Risk Level: LOW

Recommendations: 
BTC Long
ETH Hold
SOL Breakout
```

**AI 状态颜色**: `#00D4FF`

---

## 交易界面

### 布局

- Chart
- Order Panel
- AI Insights

### AI Insights 内容

```
AI Analysis

Trend Strength: Strong
Liquidity: Increasing
Risk: Medium
```

---

## 官网首页结构

### Hero Section

**主标题**: Trade Every Market with AI

**副标题**: Stocks, Crypto, Forex, Commodities - All in One AI Trading Platform

**按钮**: Start Trading

### 功能介绍（三块）

1. AI Trading Agent
2. Multi-Asset Portfolio
3. Global Market Access

### 产品界面展示

大图：Trading Dashboard

### AI 能力

- Market Regime Detection
- Signal Generation
- Risk Management
- Portfolio Optimization

### CTA

Start Your AI Trading Journey

---

## 品牌 Slogan

推荐：

1. Trade Everything with AI
2. Your AI Trading Agent
3. The Future of Trading
4. **AI-Powered Global Trading** ⭐（推荐）

---

## 品牌风格总结

**整体感觉**: OpenAI + Stripe + TradingView

但：
- 更年轻
- 更 AI

### 产品气质

**应该像**:
- Professional AI Finance

**避免**:
- 赌场 🎰
- Crypto Meme
- 赌博 UI

---

## 设计关键词（给设计师）

```
Dark AI Trading Terminal
Minimal Fintech
Futuristic SaaS
AI-first Interface
```

---

## 技术实现要求

### Tailwind CSS 配置

需要在 `tailwind.config.js` 中添加新的颜色系统：

```javascript
colors: {
  'brand': {
    primary: '#5B5CFF',    // Electric Indigo
    accent: '#00D4FF',     // Cyber Blue
    bg: {
      primary: '#0B0F1A',
      panel: '#121826',
    },
    profit: '#00C853',
    loss: '#FF3B3B',
    text: {
      primary: '#E6E8EF',
      secondary: '#8B93A7',
    }
  }
}
```

### CSS 变量

在 `index.css` 中添加：

```css
:root {
  --brand-primary: #5B5CFF;
  --brand-accent: #00D4FF;
  --brand-bg-primary: #0B0F1A;
  --brand-bg-panel: #121826;
  --brand-profit: #00C853;
  --brand-loss: #FF3B3B;
  --brand-text-primary: #E6E8EF;
  --brand-text-secondary: #8B93A7;
  
  --glow-primary: 0 0 12px rgba(91, 92, 255, 0.3);
  --glow-accent: 0 0 12px rgba(0, 212, 255, 0.35);
}
```

### 发光效果类

```css
.glow-primary {
  box-shadow: var(--glow-primary);
}

.glow-accent {
  box-shadow: var(--glow-accent);
}

.glow-border {
  position: relative;
}

.glow-border::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1px;
  background: linear-gradient(135deg, #5B5CFF, #00D4FF);
  -webkit-mask: 
    linear-gradient(#fff 0 0) content-box, 
    linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}
```

---

## 组件设计规范

### 按钮

**Primary Button**:
```css
background: #5B5CFF;
color: #FFFFFF;
box-shadow: 0 0 12px rgba(91, 92, 255, 0.3);
```

**Hover**:
```css
transform: translateY(-2px);
box-shadow: 0 4px 20px rgba(91, 92, 255, 0.5);
```

### 卡片

**Panel**:
```css
background: #121826;
border: 1px solid rgba(91, 92, 255, 0.1);
border-radius: 12px;
box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
```

### 输入框

```css
background: #0B0F1A;
border: 1px solid rgba(91, 92, 255, 0.2);
color: #E6E8EF;
font-family: 'JetBrains Mono', monospace;
```

**Focus**:
```css
border-color: #00D4FF;
box-shadow: 0 0 0 2px rgba(0, 212, 255, 0.15);
```

---

## 品牌 Slogan 应用

**主 Slogan**: AI-Powered Global Trading

**应用场景**:
- Hero Section 主标题下方
- Footer
- Meta description
- 社交媒体简介

---

## 实施优先级

### Phase 1 - 核心颜色系统
- [ ] 更新 Tailwind 配置
- [ ] 添加 CSS 变量
- [ ] 创建发光效果工具类

### Phase 2 - 基础组件
- [ ] 按钮样式
- [ ] 卡片样式
- [ ] 输入框样式
- [ ] 文本样式

### Phase 3 - 页面重构
- [ ] Landing Page Hero
- [ ] Dashboard
- [ ] AI Agent Panel
- [ ] Trading Interface

### Phase 4 - 细节优化
- [ ] 动画效果
- [ ] 响应式适配
- [ ] 性能优化

---

## 设计原则

1. **一致性**: 所有组件使用统一的颜色和间距系统
2. **可读性**: 确保文字对比度符合 WCAG 标准
3. **性能**: 使用 CSS 变量和 Tailwind 优化渲染
4. **可访问性**: 支持键盘导航和屏幕阅读器
5. **响应式**: Mobile-first 设计

---

## 注意事项

⚠️ **重要**: 
- 保持功能不变，仅调整 UI 风格
- 确保所有现有功能正常工作
- 保留原有的动画效果（如适用）
- 测试所有交易相关功能
- 确保数据展示准确性

---

## 品牌资产

### Logo 文件路径
- `/web/public/icons/nofx.svg` - 主 Logo
- 需要创建新的渐变版本

### 字体引入
已在 `index.css` 中引入：
- Inter（标题）
- JetBrains Mono（数字）

### 图标库
- Lucide React - 主要图标库
- 保持使用现有图标系统

---

## 成功标准

✅ **视觉**:
- 符合专业金融科技风格
- 展现 AI 科技感
- 年轻化但不轻浮

✅ **功能**:
- 所有功能正常工作
- 性能不下降
- 响应式布局完整

✅ **用户体验**:
- 导航清晰
- 信息层次分明
- 操作流畅

---

## 参考资源

- [Stripe Design](https://stripe.com)
- [Robinhood Design](https://robinhood.com)
- [OpenAI Design](https://openai.com)
- [TradingView](https://tradingview.com)

---

## 附录：UI 重构快速开始

### 第一步：颜色系统切换（P0）

**目标**：将现有金色主题切换为 Electric Indigo 主题

**步骤**：
1. 打开 `tailwind.config.js`
2. 替换 `nofx-gold` 为 `brand-primary: #5B5CFF`
3. 添加 `brand-accent: #00D4FF`
4. 更新背景色为 `#0B0F1A` 和 `#121826`

**验证**：
```bash
cd web
npm run dev
# 访问 http://localhost:5173
# 检查按钮、卡片、文本颜色是否正确
```

### 第二步：基础组件样式（P1）

**优先级顺序**：
1. 按钮组件（最高频使用）
2. 卡片组件（最基础布局）
3. 输入框组件（表单必需）
4. 文本样式（全局影响）

**每个组件验证**：
- Hover 效果正常
- 响应式显示正确
- 无障碍访问支持

### 第三步：核心页面重构（P2）

**页面顺序**：
1. Landing Page（门面）
2. Dashboard（最常用）
3. AI Agent Panel（核心功能）
4. 交易界面（关键功能）

**每个页面验证**：
- 所有功能正常
- 数据展示准确
- 性能无下降

### 检查清单

参考 [checklist.md](./checklist.md) 进行完整验证。

---

## 文档版本

- **Version**: 1.1.0
- **Last Updated**: 2026-03-08
- **Status**: Ready for Implementation
- **Priority**: UI-Only Refactoring (Phase 1)

---

**备注**：本文档专注于 UI 重构，不涉及功能变更。所有功能相关的改进将在后续版本中单独规划。
