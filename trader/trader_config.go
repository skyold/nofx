package trader

import (
	"nofx/store"
	"time"
)

// TraderConfig Trader 配置
// 包含交易所选择、API 密钥和策略配置
type TraderConfig struct {
	// 基本信息
	ID      string // Trader 唯一标识
	Name    string // Trader 显示名称
	AIModel string // AI 模型："qwen" 或 "deepseek"

	// 交易所选择
	Exchange   string // 交易所类型："binance", "bybit", "okx", "hyperliquid", "aster", "lighter" 等
	ExchangeID string // 交易所账户 ID（支持多账户）

	// 扫描配置
	ScanInterval time.Duration // 扫描间隔（建议 3 分钟）

	// 账户配置
	InitialBalance float64 // 初始余额（用于 P&L 计算）

	// 风控配置（仅作为提示，AI 可自主决策）
	MaxDailyLoss    float64       // 最大日亏损百分比（提示）
	MaxDrawdown     float64       // 最大回撤百分比（提示）
	StopTradingTime time.Duration // 风控触发后暂停时长

	// 仓位模式
	IsCrossMargin bool // true=全仓模式，false=逐仓模式

	// 竞赛可见性
	ShowInCompetition bool // 是否在竞赛页面显示

	// 策略配置
	StrategyConfig *store.StrategyConfig // 策略配置

	// ============================================================================
	// API 密钥配置（从 store.Exchange 获取）
	// ============================================================================

	// 通用 API 密钥
	APIKey     string // API Key
	SecretKey  string // Secret Key
	Passphrase string // Passphrase (用于 OKX 等)

	// 测试网模式
	Testnet bool // 是否使用测试网

	// Hyperliquid 专用配置
	HyperliquidWalletAddr  string // Hyperliquid 钱包地址
	HyperliquidUnifiedAcct bool   // Hyperliquid 统一账户模式
	HyperliquidPrivateKey  string // Hyperliquid 私钥

	// Aster 专用配置
	AsterUser       string // Aster 用户名
	AsterSigner     string // Aster 签名者地址
	AsterPrivateKey string // Aster 私钥

	// Lighter 专用配置
	LighterWalletAddr       string // Lighter 钱包地址
	LighterPrivateKey       string // Lighter 私钥
	LighterAPIKeyPrivateKey string // Lighter API Key 私钥
	LighterAPIKeyIndex      int    // Lighter API Key 索引
}
