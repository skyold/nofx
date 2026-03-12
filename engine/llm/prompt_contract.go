package chaos

import (
	"strings"
)

// SystemExecutionContract is the immutable protocol text for the system
const SystemExecutionContract = `
━━━━━━━━━━━━━━━━━━━━
System Execution Contract
━━━━━━━━━━━━━━━━━━━━

你是运行在自动化交易系统中的决策模块。

你必须且仅允许输出以下两个顶层结构：
<execution_reasoning> … </execution_reasoning>
<decision> … </decision>

上述结构是系统执行接口（Execution Interface）。
其字段结构、字段含义、字段合法性与执行权限
由系统代码层统一定义、校验与强制执行。

你无权：
- 新增、删除、重命名任何字段
- 输出系统未允许的 action
- 绕过系统级风险控制、仓位约束或授权裁定
- 通过解释性文本影响执行结果

你只负责：
- 基于给定数据与策略逻辑进行判断
- 填充系统接口所需的值
- 在不确定或不被授权时选择不交易

任何不符合接口规范的输出，
将被系统拒绝执行或自动降级为 NO_TRADE。

除非系统明确要求，
否则不要输出 audit_reasoning 或 debug_reasoning。
━━━━━━━━━━━━━━━━━━━━
`

// GenerateOutputSchema generates the dynamic schema specification section
// This ensures the Prompt always reflects the code-defined structures
func GenerateOutputSchema() string {
	var sb strings.Builder

	sb.WriteString(SystemExecutionContract)
	sb.WriteString("\n")
	sb.WriteString("Interface Specification (Code-Generated):\n\n")

	// 1. Reasoning Schema (Flexible Base)
	sb.WriteString("1. <execution_reasoning> Schema (Base Requirement):\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "system_risk_flag": false,  // [MANDATORY] System-wide risk fuse
  "prompt_meta": { ... },     // [OPTIONAL] Metadata
  ...                         // [STRATEGY-DEFINED] Other fields defined by the specific strategy
}`)
	sb.WriteString("\n```\n")
	sb.WriteString("Note: The detailed structure of <execution_reasoning> (e.g. opportunities, trends) is defined by the Strategy Prompt.\n\n")

	// 2. Decision Schema
	sb.WriteString("2. <decision> Schema (Strict & Immutable):\n")
	sb.WriteString("```json\n")
	sb.WriteString(`[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 5,
    "entry": 63000.00,
    "stop_loss": 62000.00,
    "take_profit": 65000.00,
    "risk_r": 0.5,
    "total_score": 85
  },
  {
    "symbol": "BTCUSDT",
    "action": "close_long"
  },
  {
    "symbol": "ETHUSDT",
    "action": "wait"
  }
]`)
	sb.WriteString("\n```\n")

	sb.WriteString("\nCritical Field Constraints:\n")
	sb.WriteString("- action: Must be one of [open_long, open_short, close_long, close_short, hold, wait]\n")
	sb.WriteString("- risk_r (open_* only): Must be a number between 0.1 and 1.5\n")
	sb.WriteString("- open_* actions: MUST include leverage, entry, stop_loss, take_profit, risk_r, total_score\n")
	sb.WriteString("- close_*/hold/wait actions: MUST include ONLY symbol and action\n")

	return sb.String()
}
