# TraderManager 迁移状态

## 已完成的工作 ✅

### 1. 核心组件实现
- ✅ `scheduler.Scheduler` 接口和实现
- ✅ `scheduler.SchedulerImpl` 带有数据保存功能
- ✅ `engine/chaos/` - Chaos Engine
- ✅ `engine/nofx/` - Nofx Engine  
- ✅ `engine/grid/` - Grid Engine
- ✅ `trader/auto_trader.go` - 简化后的 Trader 实现

### 2. TraderManager 结构修改
- ✅ 将 `traders map[string]*trader.AutoTrader` 改为 `schedulers map[string]scheduler.Scheduler`
- ✅ 更新 `TraderExecutorAdapter` 使用 Scheduler
- ✅ 添加 `scheduler` 包导入

## 待完成的工作 ⏳

### 需要修改的方法列表

`TraderManager` 有以下方法需要更新以使用 `Scheduler`：

#### 1. 生命周期管理方法
- `LoadTradersFromStore()` - 从数据库加载 Trader（需要改为创建 Scheduler）
- `AutoStartRunningTraders()` - 自动启动运行中的 Trader
- `StartAll()` - 启动所有 Trader
- `StopAll()` - 停止所有 Trader
- `StartTrader(traderID string)` - 启动指定 Trader
- `StopTrader(traderID string)` - 停止指定 Trader

#### 2. 查询方法
- `GetTrader(traderID string)` - 获取 Trader（需要返回 Scheduler 或适配后的接口）
- `GetAllTraders()` - 获取所有 Trader
- `GetRunningTraders()` - 获取运行中的 Trader
- `GetTraderStatus(traderID string)` - 获取 Trader 状态
- `GetTraderStats(traderID string)` - 获取 Trader 统计

#### 3. 配置方法
- `UpdateTraderConfig()` - 更新 Trader 配置
- `SetTraderAIModel()` - 设置 AI 模型
- 等等...

### 关键改动点

#### 1. 创建 Scheduler 而不是 AutoTrader

**旧代码**:
```go
func (tm *TraderManager) addTraderFromStore(...) error {
    // 创建 AutoTrader（保持这个调用方式）
    at, err := trader.NewAutoTrader(config, st, userID)
    if err != nil {
        return err
    }
    
    tm.traders[traderID] = at
    
    // 如果需要运行，直接调用 Run()
    if trader.IsRunning {
        go at.Run()
    }
    
    return nil
}
```

**新代码**:
```go
func (tm *TraderManager) addTraderFromStore(...) error {
    // 1. 创建 AutoTrader（封装了交易所创建逻辑，保持向后兼容）
    autoTrader, err := trader.NewAutoTrader(config, st, userID)
    if err != nil {
        return err
    }
    
    // 2. 创建 Engine（根据策略类型选择）
    var engine scheduler.Engine
    if config.StrategyType == "chaos" {
        engine = chaos.NewChaosEngine(config.StrategyConfig)
        engine.SetDependencies(autoTrader, strategyEngine, st, traderID, startTime, callCount)
    } else if config.StrategyType == "grid" {
        // Grid Engine 实现
    } else if config.StrategyType == "nofx" {
        // Nofx Engine 实现
    }
    
    // 3. 创建 Scheduler
    sched := scheduler.NewScheduler(config.ScanInterval)
    sched.SetEngine(engine)
    sched.SetTrader(autoTrader)
    sched.SetStore(st)
    
    // 4. 保存 Scheduler
    tm.schedulers[traderID] = sched
    
    // 5. 如果需要运行，调用 Scheduler.Start()
    if trader.IsRunning {
        go sched.Start()
    }
    
    return nil
}
```

**关键点**:
- ✅ `trader.NewAutoTrader(config, st, userID)` - 保持旧的调用方式
- ✅ AutoTrader 内部根据 `config.Exchange` 自动创建对应的交易所 trader
- ✅ 调用代码不需要关心具体的交易所实现
- ✅ 符合封装原则和向后兼容性

#### 2. 启动/停止方法

**旧代码**:
```go
func (tm *TraderManager) StartAll() {
    for _, t := range tm.traders {
        go t.Run()
    }
}

func (tm *TraderManager) StopAll() {
    for _, t := range tm.traders {
        t.Stop()
    }
}
```

**新代码**:
```go
func (tm *TraderManager) StartAll() {
    for _, sched := range tm.schedulers {
        go sched.Start()
    }
}

func (tm *TraderManager) StopAll() {
    for _, sched := range tm.schedulers {
        sched.Stop()
    }
}
```

#### 3. 查询方法

**旧代码**:
```go
func (tm *TraderManager) GetTrader(traderID string) (*trader.AutoTrader, error) {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    t, ok := tm.traders[traderID]
    if !ok {
        return nil, fmt.Errorf("trader not found: %s", traderID)
    }
    
    return t, nil
}
```

**新代码**:
```go
func (tm *TraderManager) GetScheduler(traderID string) (scheduler.Scheduler, error) {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    sched, ok := tm.schedulers[traderID]
    if !ok {
        return nil, fmt.Errorf("scheduler not found: %s", traderID)
    }
    
    return sched, nil
}
```

### 影响范围

需要同时修改的文件：

1. **API 层** (`api/*.go`)
   - 所有调用 `GetTrader()` 的地方
   - 所有访问 `AutoTrader` 字段的地方

2. **Debate 模块** (`debate/*.go`)
   - `TraderExecutorAdapter` 的实现

3. **测试文件** (`manager/trader_manager_test.go`)
   - 所有测试用例

### 迁移策略

由于改动较大，建议采用**渐进式迁移**：

#### 阶段 1：并行运行（1-2 天）
```go
type TraderManager struct {
    // 旧的 AutoTrader（用于向后兼容的 API）
    traders map[string]*trader.AutoTrader
    
    // 新的 Scheduler（用于实际运行）
    schedulers map[string]scheduler.Scheduler
}
```

#### 阶段 2：逐步迁移（3-5 天）
- 新创建的 Trader 使用 Scheduler
- 现有的 Trader 保持使用 AutoTrader
- 逐步测试和修复

#### 阶段 3：完全迁移（2-3 天）
- 删除 `traders` 字段
- 所有方法都使用 `schedulers`
- 删除旧的 AutoTrader 相关代码

### 当前状态

- ✅ `TraderManager` 结构已修改为使用 `schedulers`
- ⏳ 所有方法还在使用旧的 `traders` 字段（需要更新）
- ⏳ API 层和测试还未更新

### 下一步

1. 完成 `addTraderFromStore()` 方法的重写
2. 更新所有生命周期管理方法
3. 更新所有查询方法
4. 更新 API 层
5. 更新测试
6. 验证编译和运行

### 预计工作量

- 核心代码修改：2-3 天
- API 层更新：1-2 天
- 测试和修复：2-3 天
- **总计**: 5-8 天
