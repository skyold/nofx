# 引擎接口迁移状态

## ✅ 已完成

### 1. Engine 接口定义
- ✅ `engine/interface.go` - 定义了标准的 Engine 接口
- ✅ 6 个核心方法：
  - `Name()`
  - `BuildContext()`
  - `BuildSystemPrompt()`
  - `BuildUserPrompt()`
  - `CallLLM()`
  - `ParseResponse()`
  - `ValidateDecisions()`

### 2. ChaosEngine 部分实现
- ✅ 添加了 `Name()` 方法
- ✅ 添加了 `BuildContext()` 方法框架
- ⚠️ `BuildSystemPrompt()` - 有重复声明冲突
- ⚠️ `BuildUserPrompt()` - 有重复声明冲突
- ⚠️ `ValidateDecisions()` - 有重复声明冲突

## ❌ 存在的问题

### 1. 方法重复声明

当前 `chaos/engine.go` 中存在：
- 旧的 `BuildSystemPrompt(accountEquity float64, variant string)` 方法
- 新的 `BuildSystemPrompt(ctx *engine.Context)` 方法（Engine 接口要求）

**冲突原因**：方法名相同但签名不同

### 2. 类型不匹配

旧的代码使用：
```go
[]chaos.Decision
*chaos.Reasoning
*chaos.ChaosContext
```

新的 Engine 接口要求：
```go
[]engine.Decision
*engine.Context
```

### 3. 循环依赖

- `chaos` 包需要导入 `engine` 包
- 但 `engine.Context` 需要包含 `chaos.ChaosContext` 的数据
- 导致类型转换复杂

## 🔧 解决方案

### 方案 A：重命名旧方法（推荐）

保留旧方法，但重命名以避免冲突：

```go
// 旧方法重命名，加 Legacy 前缀
func (e *ChaosEngine) LegacyBuildSystemPrompt(accountEquity float64, variant string) string {
    // 原有实现
}

// 新方法实现接口
func (e *ChaosEngine) BuildSystemPrompt(ctx *engine.Context) string {
    // 新实现
}
```

### 方案 B：完全替换（破坏性）

删除所有旧方法，只保留新接口：
```go
// 删除所有旧的 BuildSystemPrompt, BuildUserPrompt, ValidateDecisions
// 只保留 Engine 接口的实现
```

### 方案 C：适配器模式（过渡）

创建适配器层：
```go
type ChaosEngineAdapter struct {
    *ChaosEngine
}

func (a *ChaosEngineAdapter) BuildSystemPrompt(ctx *engine.Context) string {
    // 转换为旧格式
    chaosCtx := convertEngineContextToChaosContext(ctx)
    return a.ChaosEngine.BuildSystemPromptWithContext(chaosCtx)
}
```

## 📋 下一步行动

### 立即需要修复

1. **解决重复声明** - 重命名或删除旧方法
2. **类型转换** - 实现 `engine.Context` ↔ `ChaosContext` 转换
3. **方法签名对齐** - 确保与 Engine 接口一致

### 后续工作

1. **完善 BuildContext()** - 迁移 AutoTrader.buildChaosContext() 逻辑
2. **实现 CallLLM()** - 封装 MCP Client 调用
3. **实现 ParseResponse()** - 复用现有解析逻辑
4. **实现 ValidateDecisions()** - 迁移现有验证逻辑

## 🎯 架构关系

```
engine.Engine (接口)
    ↓ 实现
chaos.ChaosEngine
    ↓ 使用
chaos.ContextBuilder
    ↓ 构建
engine.Context (通用)
    ↓ 转换为
chaos.ChaosContext (特定)
```

## ⚠️ 注意事项

1. **向后兼容** - 现有调用 `BuildSystemPrompt()` 的代码需要更新
2. **测试** - 所有修改都需要测试验证
3. **文档** - 需要更新 API 文档
