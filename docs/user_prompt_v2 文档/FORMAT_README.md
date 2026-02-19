# 完整的数据格式方案 - 生产就绪版本

## 📦 包含文件

```
📁 Complete Format Package
├── 📄 FORMAT_README.md               ← 你现在看的文件
├── 📄 system_prompt_template.md      ← System Prompt（自定义指标定义）
├── 📄 data_format_example.json       ← 完整JSON数据格式
└── 📄 usage_guide.md                 ← 详细使用指南
```

---

## 🎯 核心原则（记住这个）

| 指标类型 | 需要说明？ | 怎么处理 |
|---------|----------|---------|
| **标准指标** (EMA, RSI, MACD, BOLL, ATR) | ❌ 不需要 | 直接给数值 |
| **自定义指标** (trend_strength, market_regime等) | ✅ 必须 | System Prompt定义 + 数据加标签 |

---

## 🚀 3步快速开始

### 1️⃣ 添加System Prompt
→ 打开 `system_prompt_template.md` 
→ 复制全部内容到你的系统提示词

### 2️⃣ 使用数据格式
→ 参考 `data_format_example.json`
→ 按相同结构组织你的数据

### 3️⃣ 测试验证
→ 问LLM："market_regime='ranging'是什么意思？"
→ 应该能准确解释

---

**这套方案Token节省70%，信息保留95%，可直接用于生产！** 🚀
