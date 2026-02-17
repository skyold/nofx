// =============================================================================
// Chaos Trading System - User Prompt Entry Point
// =============================================================================
// 
// User Prompt 是发送给 LLM 的用户提示词的一部分。
// 完整提示词 = System Prompt + User Prompt
//
// 此文件作为 User Prompt 的入口，根据配置选择对应版本：
// - legacy/v1: 文本格式 (user_prompt_legacy.go)
// - v2: JSON格式 (user_prompt_v2.go) - 默认版本
// - v4: 结构事实版 (user_prompt_v4.go)
//
// 与 system_prompt.go 对应，engine.go 只需调用 BuildUserPromptFromChaosContext
//
// 版本切换：通过配置 prompt_version 字段切换
// =============================================================================

package chaos

// BuildUserPromptFromChaosContext builds User Prompt using ChaosContext.
// This is the entry point that selects the appropriate version based on config.
//
// 版本选择逻辑：
//   - "v1" 或 "legacy" → buildUserPromptLegacy (文本格式)
//   - "v2" (默认)     → buildUserPromptV2 (JSON格式)
//   - "v4"            → buildUserPromptV4 (结构事实版)
func (e *ChaosEngine) BuildUserPromptFromChaosContext(ctx *ChaosContext) string {
	version := "v2" // default to v2 (stable)
	if ctx.Config != nil && ctx.Config.PromptVersion != "" {
		version = ctx.Config.PromptVersion
	}

	switch version {
	case "v1", "legacy":
		return e.buildUserPromptLegacy(ctx)
	case "v4":
		return e.buildUserPromptV4(ctx)
	case "v2":
		fallthrough
	default:
		return e.buildUserPromptV2(ctx)
	}
}
