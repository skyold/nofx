# 迁移计划文档 - 阶段 1

## 1. 迁移目标

将当前的交易系统从**混合架构**迁移到**三层架构**（引擎 - 调度器 - Trader），实现：
- ✅ 职责清晰：引擎思考、Trader 执行、调度器协调
- ✅ 易于扩展：新增引擎/交易所更简单
- ✅ 易于测试：各模块独立测试
- ✅ 易于维护：降低代码复杂度

---

## 2. 迁移原则

### 2.1 核心原则

1. **渐进式重构**
   - 不破坏现有功能
   - 每个阶段独立验证
   - 可随时回滚

2. **测试先行**
   - 先写测试再重构
   - 保持测试覆盖率 > 80%
   - 回归测试必须通过

3. **文档驱动**
   - 先写文档再写代码
   - 代码注释完整
   - API 文档同步更新

4. **向后兼容**
   - 尽可能保持 API 兼容
   - 提供适配层
   - 提供迁移脚本

### 2.2 不做的改变

- ❌ 不修改数据库 schema
- ❌ 不修改业务逻辑
- ❌ 不修改外部 API
- ❌ 不修改配置格式（阶段 1-4）

---

## 3. 迁移时间表

### 总体时间线

```
Week 1: 阶段 1 - 接口设计
Week 2: 阶段 2 - 上下文构建迁移
Week 3: 阶段 3 - 执行器迁移
Week 4: 阶段 4 - 调度器简化
Week 5: 阶段 5 - 接口统一和测试
```

### 详细里程碑

| 阶段 | 开始日期 | 结束日期 | 交付物 |
|------|----------|----------|--------|
| 阶段 1 | Day 1 | Day 2 | 接口文件、文档 |
| 阶段 2 | Day 3 | Day 6 | ContextBuilder、测试 |
| 阶段 3 | Day 7 | Day 9 | Executor、测试 |
| 阶段 4 | Day 10 | Day 13 | Scheduler、测试 |
| 阶段 5 | Day 14 | Day 17 | 完整测试、文档 |

---

## 4. 阶段 1：文档化和接口设计

### 4.1 目标

- [ ] 定义清晰的 Engine、Trader、Scheduler 接口
- [ ] 评估接口设计的合理性
- [ ] 评估与当前实现的差异
- [ ] 制定详细的迁移计划

### 4.2 任务分解

#### Task 1.1: 创建 Engine 接口

**负责人**：后端开发
**预计时间**：2 小时
**依赖**：无

**步骤**：
1. 创建 `engine/interface.go`
2. 定义 `Engine` 接口
3. 定义 `Context`、`Decision`、`ValidatedDecision` 等类型
4. 编写接口文档注释
5. 代码审查

**验收标准**：
- [ ] 接口文件创建完成
- [ ] 所有方法有完整注释
- [ ] 通过代码审查
- [ ] 无编译错误

**代码示例**：
```go
// engine/interface.go
package engine

import "context"

type Engine interface {
    Name() string
    BuildContext(ctx context.Context, runtime RuntimeInfo) (*Context, error)
    BuildSystemPrompt(ctx *Context) string
    BuildUserPrompt(ctx *Context) string
    CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*AIResponse, error)
    ParseResponse(response *AIResponse) ([]Decision, error)
    ValidateDecisions(ctx context.Context, decisions []Decision, context *Context) ([]ValidatedDecision, error)
}
```

---

#### Task 1.2: 创建 Trader 接口（增强版）

**负责人**：后端开发
**预计时间**：2 小时
**依赖**：Task 1.1

**步骤**：
1. 创建 `trader/types.go`
2. 定义 `AccountInfo`、`PositionInfo`、`OrderResult` 等类型
3. 增强 `trader/interface.go`
4. 添加 `ExecuteDecision()` 方法
5. 所有方法增加 `context.Context` 参数
6. 编写文档注释

**验收标准**：
- [ ] 类型文件创建完成
- [ ] 接口增强完成
- [ ] 所有方法有 context 参数
- [ ] 注释完整

**代码示例**：
```go
// trader/interface.go
type Trader interface {
    GetExchange() string
    GetAccountInfo(ctx context.Context) (*AccountInfo, error)
    GetPositions(ctx context.Context) ([]PositionInfo, error)
    // ... 其他方法
    ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*OrderResult, error)
}
```

---

#### Task 1.3: 创建 Scheduler 接口

**负责人**：后端开发
**预计时间**：2 小时
**依赖**：Task 1.1, Task 1.2

**步骤**：
1. 创建 `scheduler/interface.go`
2. 定义 `Scheduler` 接口
3. 定义 `SchedulerStatus`、`SchedulerStats` 类型
4. 编写文档注释

**验收标准**：
- [ ] 接口文件创建完成
- [ ] 状态类型定义完整
- [ ] 注释完整

**代码示例**：
```go
// scheduler/interface.go
type Scheduler interface {
    Start() error
    Stop() error
    IsRunning() bool
    SetEngine(engine engine.Engine)
    SetTrader(trader trader.Trader)
    SetInterval(interval time.Duration)
    GetStatus() *SchedulerStatus
    GetStats() *SchedulerStats
}
```

---

#### Task 1.4: 编写影响分析文档

**负责人**：技术负责人
**预计时间**：4 小时
**依赖**：Task 1.1-1.3

**步骤**：
1. 分析受影响的模块
2. 识别需要修改的文件
3. 评估风险
4. 制定缓解措施
5. 编写文档

**验收标准**：
- [ ] 影响分析完整
- [ ] 风险识别准确
- [ ] 缓解措施可行
- [ ] 文档通过评审

---

#### Task 1.5: 编写迁移计划文档

**负责人**：技术负责人
**预计时间**：2 小时
**依赖**：Task 1.4

**步骤**：
1. 制定迁移时间表
2. 分解任务
3. 估算工作量
4. 编写文档

**验收标准**：
- [ ] 时间表合理
- [ ] 任务分解清晰
- [ ] 工作量估算准确
- [ ] 文档通过评审

---

#### Task 1.6: 接口设计评审

**负责人**：全体团队成员
**预计时间**：2 小时
**依赖**：Task 1.1-1.5

**步骤**：
1. 准备评审材料
2. 召评审会议
3. 记录问题
4. 修改完善

**验收标准**：
- [ ] 评审会议完成
- [ ] 问题记录完整
- [ ] 修改完成
- [ ] 评审通过

---

### 4.3 交付物清单

| 文件 | 状态 | 负责人 | 截止日期 |
|------|------|--------|----------|
| `engine/interface.go` | ⏳ 待开始 | 后端开发 | Day 1 |
| `engine/types.go` | ⏳ 待开始 | 后端开发 | Day 1 |
| `trader/types.go` | ⏳ 待开始 | 后端开发 | Day 1 |
| `trader/interface.go` | ⏳ 待开始 | 后端开发 | Day 1 |
| `scheduler/interface.go` | ⏳ 待开始 | 后端开发 | Day 2 |
| `scheduler/types.go` | ⏳ 待开始 | 后端开发 | Day 2 |
| `spec/interface_design.md` | ⏳ 待开始 | 后端开发 | Day 2 |
| `spec/impact_analysis.md` | ⏳ 待开始 | 技术负责人 | Day 2 |
| `spec/migration_plan.md` | ⏳ 待开始 | 技术负责人 | Day 2 |

---

### 4.4 验收流程

```
1. 开发者自测
   ↓
2. 代码审查（至少 2 人）
   ↓
3. 运行编译
   ↓
4. 运行现有测试（确保不破坏）
   ↓
5. 技术负责人审批
   ↓
6. 合并到主分支
```

---

## 5. 阶段 2：迁移上下文构建到引擎

### 5.1 目标

将 `buildChaosContext()` 从 `AutoTrader` 迁移到 `ChaosEngine`，使引擎负责完整的上下文构建。

### 5.2 任务分解

#### Task 2.1: 创建 Chaos ContextBuilder

**负责人**：后端开发
**预计时间**：4 小时
**依赖**：阶段 1 完成

**步骤**：
1. 创建 `chaos/context_builder.go`
2. 定义 `ContextBuilder` 结构
3. 实现 `BuildContext()` 方法
4. 迁移 `buildChaosContext()` 逻辑
5. 编写单元测试

**代码结构**：
```go
// chaos/context_builder.go
type ContextBuilder struct {
    trader       trader.Trader
    marketClient *market.Client
    nofxosClient *nofxos.Client
    config       *ChaosConfig
}

func (cb *ContextBuilder) BuildContext(ctx context.Context, runtime RuntimeInfo) (*engine.Context, error) {
    // 1. 获取账户信息
    account, err := cb.trader.GetAccountInfo(ctx)
    
    // 2. 获取持仓信息
    positions, err := cb.trader.GetPositions(ctx)
    
    // 3. 获取候选币种
    candidateCoins, err := cb.GetCandidateCoins()
    
    // 4. 获取市场数据
    marketDataMap, err := cb.fetchMarketData(candidateCoins)
    
    // 5. 获取排名数据
    oiRanking, _ := cb.nofxosClient.GetOIRanking()
    
    // 6. 组装上下文
    return &engine.Context{
        Account:        account,
        Positions:      positions,
        CandidateCoins: candidateCoins,
        MarketDataMap:  marketDataMap,
        OIRankingData:  oiRanking,
        // ...
    }, nil
}
```

**验收标准**：
- [ ] ContextBuilder 实现完成
- [ ] BuildContext() 方法正确
- [ ] 单元测试覆盖率 > 80%
- [ ] 无编译错误

---

#### Task 2.2: 创建 Kernel ContextBuilder

**负责人**：后端开发
**预计时间**：4 小时
**依赖**：阶段 1 完成

**步骤**：
1. 创建 `kernel/context_builder.go`
2. 定义 `ContextBuilder` 结构
3. 实现 `BuildContext()` 方法
4. 迁移现有逻辑
5. 编写单元测试

**验收标准**：
- [ ] ContextBuilder 实现完成
- [ ] 单元测试通过
- [ ] 代码符合规范

---

#### Task 2.3: 修改 ChaosEngine

**负责人**：后端开发
**预计时间**：4 小时
**依赖**：Task 2.1

**步骤**：
1. 修改 `ChaosEngine` 结构，添加 `builder` 字段
2. 实现 `BuildContext()` 方法（委托给 builder）
3. 修改 `BuildSystemPromptWithContext()` 为 `BuildSystemPrompt()`
4. 修改方法签名适配 Engine 接口
5. 添加 `CallLLM()` 方法
6. 更新所有测试

**代码变更**：
```go
// chaos/engine.go
type ChaosEngine struct {
    builder   *ContextBuilder
    config    *ChaosConfig
    mcpClient mcp.Client
}

func (ce *ChaosEngine) BuildContext(ctx context.Context, runtime RuntimeInfo) (*engine.Context, error) {
    return ce.builder.BuildContext(ctx, runtime)
}

func (ce *ChaosEngine) BuildSystemPrompt(ctx *engine.Context) string {
    // 调用 builder 的私有方法
    return ce.builder.buildSystemPrompt(ctx)
}

func (ce *ChaosEngine) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*engine.AIResponse, error) {
    return ce.mcpClient.CallWithMessages(systemPrompt, userPrompt)
}
```

**验收标准**：
- [ ] 实现 Engine 接口
- [ ] 所有方法签名正确
- [ ] 测试通过

---

#### Task 2.4: 修改 StrategyEngine

**负责人**：后端开发
**预计时间**：4 小时
**依赖**：Task 2.2

**步骤**：
1. 修改 `StrategyEngine` 结构
2. 实现 `BuildContext()` 方法
3. 修改 `BuildSystemPrompt()` 和 `BuildUserPrompt()` 签名
4. 添加 `CallLLM()` 方法
5. 更新测试

**验收标准**：
- [ ] 实现 Engine 接口
- [ ] 测试通过

---

#### Task 2.5: 修改 AutoTrader.RunChaosCycle()

**负责人**：后端开发
**预计时间**：2 小时
**依赖**：Task 2.3

**步骤**：
1. 修改 `RunChaosCycle()` 调用 `engine.BuildContext()`
2. 删除 `buildChaosContext()` 方法
3. 简化代码
4. 更新测试

**代码变更**：
```go
// trader/auto_trader_chaos.go
func (at *AutoTrader) RunChaosCycle() error {
    // 之前：ctx, err := at.buildChaosContext()
    // 现在：
    runtime := engine.RuntimeInfo{
        CurrentTime: time.Now(),
        // ...
    }
    ctx, err := at.chaosEngine.BuildContext(context.Background(), runtime)
    if err != nil {
        return err
    }
    
    // 其他逻辑不变
}
```

**验收标准**：
- [ ] 调用新接口
- [ ] 功能正常
- [ ] 测试通过

---

#### Task 2.6: 集成测试

**负责人**：测试工程师
**预计时间**：4 小时
**依赖**：Task 2.1-2.5

**步骤**：
1. 编写集成测试
2. 测试完整流程
3. 记录问题
4. 修复 bug

**验收标准**：
- [ ] 集成测试通过
- [ ] 无严重 bug
- [ ] 性能无回退

---

### 5.3 交付物清单

| 文件 | 状态 | 负责人 | 截止日期 |
|------|------|--------|----------|
| `chaos/context_builder.go` | ⏳ 待开始 | 后端开发 | Day 4 |
| `kernel/context_builder.go` | ⏳ 待开始 | 后端开发 | Day 4 |
| `chaos/engine.go` (修改) | ⏳ 待开始 | 后端开发 | Day 5 |
| `kernel/engine.go` (修改) | ⏳ 待开始 | 后端开发 | Day 5 |
| `trader/auto_trader_chaos.go` (修改) | ⏳ 待开始 | 后端开发 | Day 6 |
| 单元测试 | ⏳ 待开始 | 后端开发 | Day 6 |
| 集成测试 | ⏳ 待开始 | 测试工程师 | Day 6 |

---

## 6. 阶段 3：迁移执行器到 Trader 层

### 6.1 目标

将 `ChaosExecutor` 从 `chaos/` 包迁移到 `trader/` 包，使执行器属于 Trader 层。

### 6.2 关键任务

1. 创建 `trader/executor.go`
2. 迁移 `chaos/executor.go` 内容
3. 修改 `ChaosExecutor` 接收 `Trader` 接口
4. 删除 `chaos/executor.go`
5. 更新 `AutoTrader.executeChaosDecision()`
6. 编写测试

### 6.3 时间表

| 任务 | 预计时间 | 负责人 |
|------|----------|--------|
| 创建 Executor | 4 小时 | 后端开发 |
| 迁移代码 | 2 小时 | 后端开发 |
| 修改依赖 | 2 小时 | 后端开发 |
| 删除旧文件 | 1 小时 | 后端开发 |
| 更新测试 | 4 小时 | 测试工程师 |
| **总计** | **13 小时** | |

---

## 7. 阶段 4：简化调度器

### 7.1 目标

将 `AutoTrader` 简化为纯调度器，移除所有业务逻辑。

### 7.2 关键任务

1. 创建 `trader/scheduler.go`
2. 定义 `Scheduler` 接口
3. 迁移调度逻辑
4. 简化 `AutoTrader`
5. 编写测试

### 7.3 时间表

| 任务 | 预计时间 | 负责人 |
|------|----------|--------|
| 创建 Scheduler | 6 小时 | 后端开发 |
| 迁移逻辑 | 4 小时 | 后端开发 |
| 简化 AutoTrader | 2 小时 | 后端开发 |
| 更新测试 | 4 小时 | 测试工程师 |
| **总计** | **16 小时** | |

---

## 8. 阶段 5：接口统一和测试

### 8.1 目标

定义统一的 Engine、Trader、Scheduler 接口，并完成全面测试。

### 8.2 关键任务

1. 创建 `engine/interface.go`（如果阶段 1 未创建）
2. 创建 `trader/interface.go`（增强版）
3. 创建 `scheduler/interface.go`
4. 编写接口一致性测试
5. 编写集成测试
6. 性能基准测试
7. 文档完善

### 8.3 时间表

| 任务 | 预计时间 | 负责人 |
|------|----------|--------|
| 接口文件创建 | 4 小时 | 后端开发 |
| 接口测试 | 6 小时 | 测试工程师 |
| 集成测试 | 8 小时 | 测试工程师 |
| 性能测试 | 4 小时 | 测试工程师 |
| 文档完善 | 4 小时 | 后端开发 |
| **总计** | **26 小时** | |

---

## 9. 质量保证

### 9.1 代码审查清单

- [ ] 代码符合 Go 规范
- [ ] 函数长度 < 50 行
- [ ] 圈复杂度 < 10
- [ ] 注释完整
- [ ] 无硬编码
- [ ] 错误处理完整
- [ ] 无 panic
- [ ] 资源正确关闭
- [ ] 无数据竞争
- [ ] 单元测试完整

### 9.2 测试要求

**单元测试**：
- 覆盖率 > 80%
- 所有公共方法都有测试
- 边界条件测试
- 错误处理测试

**集成测试**：
- 完整流程测试
- 异常场景测试
- 性能测试

**回归测试**：
- 现有测试全部通过
- 无破坏性变更

### 9.3 性能要求

- 单次循环耗时 < 5 秒
- 内存占用 < 100MB
- Goroutine 数量 < 20
- 无内存泄漏

---

## 10. 风险管理

### 10.1 风险矩阵

| 风险 | 影响 | 概率 | 优先级 | 缓解措施 |
|------|------|------|--------|----------|
| 破坏现有功能 | 高 | 中 | P0 | 全面测试 |
| 进度延迟 | 中 | 中 | P1 | 每日站会 |
| 人员不足 | 中 | 低 | P2 | 调整优先级 |
| 技术难点 | 高 | 低 | P1 | 技术预研 |

### 10.2 应急预案

**如果阶段 2 失败**：
1. 回滚到阶段 1
2. 分析问题
3. 调整方案
4. 重新实施

**如果阶段 3 失败**：
1. 保留 `chaos/executor.go`
2. 暂时不删除
3. 继续其他阶段
4. 后续再解决

**如果阶段 4 失败**：
1. 保留 `AutoTrader` 现有逻辑
2. 不简化
3. 继续阶段 5
4. 后续再优化

---

## 11. 沟通计划

### 11.1 会议安排

| 会议 | 频率 | 参与者 | 时长 |
|------|------|--------|------|
| 每日站会 | 每天 | 全体 | 15 分钟 |
| 周进度评审 | 每周 | 全体 | 1 小时 |
| 技术评审 | 按需 | 技术团队 | 2 小时 |
| 阶段评审 | 每阶段结束 | 全体 | 2 小时 |

### 11.2 沟通渠道

- **即时通讯**：Slack/钉钉
- **邮件**：重要决策
- **文档**：Confluence/Notion
- **代码审查**：GitHub/GitLab

### 11.3 报告机制

- **日报**：每日站会同步
- **周报**：周五发送
- **阶段报告**：阶段结束时发送
- **总结报告**：项目结束时发送

---

## 12. 成功标准

### 12.1 技术指标

- [ ] 接口测试覆盖率 100%
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] 性能无回退
- [ ] 无严重 bug

### 12.2 业务指标

- [ ] 功能完整
- [ ] 无破坏性变更
- [ ] 用户无感知
- [ ] 数据一致

### 12.3 团队指标

- [ ] 代码审查通过
- [ ] 文档完整
- [ ] 知识传递完成
- [ ] 团队满意度 > 80%

---

## 13. 后续工作

### 13.1 短期计划（1-2 个月）

1. **多引擎支持**
   - 支持同时运行 Chaos 和 Kernel
   - 决策融合机制

2. **风控模块**
   - 最大回撤控制
   - 仓位控制
   - 交易频率限制

3. **监控告警**
   - 指标收集
   - 告警规则
   - 可视化

### 13.2 中期计划（3-6 个月）

1. **插件化架构**
   - 插件接口
   - 插件市场
   - 插件管理

2. **云原生支持**
   - Kubernetes 部署
   - 自动扩缩容
   - 服务网格

3. **AI 模型优化**
   - 多模型支持
   - 模型热切换
   - 模型评估

### 13.3 长期计划（6-12 个月）

1. **分布式架构**
   - 分布式调度
   - 分布式存储
   - 分布式计算

2. **机器学习平台**
   - 特征工程
   - 模型训练
   - 模型部署

3. **生态建设**
   - 开发者社区
   - 文档完善
   - 最佳实践

---

## 附录：模板和检查清单

### A. 代码审查检查清单

```markdown
## 代码审查清单

### 代码质量
- [ ] 代码格式正确
- [ ] 命名规范
- [ ] 函数简洁
- [ ] 注释完整

### 功能正确性
- [ ] 逻辑正确
- [ ] 边界条件处理
- [ ] 错误处理完整
- [ ] 无 panic

### 性能
- [ ] 无性能问题
- [ ] 无内存泄漏
- [ ] 资源正确关闭

### 测试
- [ ] 单元测试完整
- [ ] 测试用例充分
- [ ] 测试通过

### 安全
- [ ] 无安全漏洞
- [ ] 无敏感信息
- [ ] 权限控制正确
```

### B. 测试用例模板

```go
func TestEngine_BuildContext(t *testing.T) {
    tests := []struct {
        name    string
        input   RuntimeInfo
        want    *Context
        wantErr bool
    }{
        {
            name:  "normal case",
            input: RuntimeInfo{CurrentTime: time.Now()},
            want:  &Context{},
            wantErr: false,
        },
        {
            name:  "error case",
            input: RuntimeInfo{},
            want:  nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### C. 文档模板

```markdown
# 模块名称

## 概述

简要描述模块的功能和职责。

## 接口定义

```go
type Interface interface {
    Method() error
}
```

## 使用示例

```go
engine := NewEngine()
ctx, err := engine.BuildContext()
```

## 注意事项

- 注意点 1
- 注意点 2

## 相关文件

- [相关文件链接]()
```

---

## 文档历史

| 版本 | 日期 | 作者 | 变更说明 |
|------|------|------|----------|
| v1.0 | 2026-03-11 | AI Assistant | 初始版本 |
