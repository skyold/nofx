import {
  LayoutDashboard,
  Settings,
  TrendingUp,
  Clock,
  List,
  FlaskConical,
  Swords,
  Brain,
  BarChart3,
  Database,
  Store,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export interface FAQItem {
  id: string
  questionKey: string
  answerKey: string
}

export interface FAQCategory {
  id: string
  titleKey: string
  icon: LucideIcon
  items: FAQItem[]
}

export const faqCategories: FAQCategory[] = [
  {
    id: 'dashboard',
    titleKey: 'faqCategoryDashboard',
    icon: LayoutDashboard,
    items: [
      {
        id: 'dashboard-intro',
        questionKey: 'faqDashboardIntro',
        answerKey: 'faqDashboardIntroAnswer',
      },
      {
        id: 'dashboard-positions',
        questionKey: 'faqDashboardPositions',
        answerKey: 'faqDashboardPositionsAnswer',
      },
      {
        id: 'dashboard-decisions',
        questionKey: 'faqDashboardDecisions',
        answerKey: 'faqDashboardDecisionsAnswer',
      },
      {
        id: 'dashboard-equity',
        questionKey: 'faqDashboardEquity',
        answerKey: 'faqDashboardEquityAnswer',
      },
    ],
  },
  {
    id: 'config',
    titleKey: 'faqCategoryConfig',
    icon: Settings,
    items: [
      {
        id: 'config-intro',
        questionKey: 'faqConfigIntro',
        answerKey: 'faqConfigIntroAnswer',
      },
      {
        id: 'config-ai-models',
        questionKey: 'faqConfigAIModels',
        answerKey: 'faqConfigAIModelsAnswer',
      },
      {
        id: 'config-exchanges',
        questionKey: 'faqConfigExchanges',
        answerKey: 'faqConfigExchangesAnswer',
      },
      {
        id: 'config-traders',
        questionKey: 'faqConfigTraders',
        answerKey: 'faqConfigTradersAnswer',
      },
    ],
  },
  {
    id: 'strategy',
    titleKey: 'faqCategoryStrategy',
    icon: Brain,
    items: [
      {
        id: 'strategy-intro',
        questionKey: 'faqStrategyIntro',
        answerKey: 'faqStrategyIntroAnswer',
      },
      {
        id: 'strategy-coin-source',
        questionKey: 'faqStrategyCoinSource',
        answerKey: 'faqStrategyCoinSourceAnswer',
      },
      {
        id: 'strategy-indicators',
        questionKey: 'faqStrategyIndicators',
        answerKey: 'faqStrategyIndicatorsAnswer',
      },
      {
        id: 'strategy-risk',
        questionKey: 'faqStrategyRisk',
        answerKey: 'faqStrategyRiskAnswer',
      },
    ],
  },
  {
    id: 'backtest',
    titleKey: 'faqCategoryBacktest',
    icon: FlaskConical,
    items: [
      {
        id: 'backtest-intro',
        questionKey: 'faqBacktestIntro',
        answerKey: 'faqBacktestIntroAnswer',
      },
      {
        id: 'backtest-setup',
        questionKey: 'faqBacktestSetup',
        answerKey: 'faqBacktestSetupAnswer',
      },
      {
        id: 'backtest-results',
        questionKey: 'faqBacktestResults',
        answerKey: 'faqBacktestResultsAnswer',
      },
    ],
  },
  {
    id: 'arena',
    titleKey: 'faqCategoryArena',
    icon: Swords,
    items: [
      {
        id: 'arena-intro',
        questionKey: 'faqArenaIntro',
        answerKey: 'faqArenaIntroAnswer',
      },
      {
        id: 'arena-debate',
        questionKey: 'faqArenaDebate',
        answerKey: 'faqArenaDebateAnswer',
      },
      {
        id: 'arena-consensus',
        questionKey: 'faqArenaConsensus',
        answerKey: 'faqArenaConsensusAnswer',
      },
    ],
  },
  {
    id: 'leaderboard',
    titleKey: 'faqCategoryLeaderboard',
    icon: BarChart3,
    items: [
      {
        id: 'leaderboard-intro',
        questionKey: 'faqLeaderboardIntro',
        answerKey: 'faqLeaderboardIntroAnswer',
      },
      {
        id: 'leaderboard-metrics',
        questionKey: 'faqLeaderboardMetrics',
        answerKey: 'faqLeaderboardMetricsAnswer',
      },
    ],
  },
  {
    id: 'time-machine',
    titleKey: 'faqCategoryTimeMachine',
    icon: Clock,
    items: [
      {
        id: 'time-machine-intro',
        questionKey: 'faqTimeMachineIntro',
        answerKey: 'faqTimeMachineIntroAnswer',
      },
      {
        id: 'time-machine-usage',
        questionKey: 'faqTimeMachineUsage',
        answerKey: 'faqTimeMachineUsageAnswer',
      },
    ],
  },
  {
    id: 'transactions',
    titleKey: 'faqCategoryTransactions',
    icon: List,
    items: [
      {
        id: 'transactions-intro',
        questionKey: 'faqTransactionsIntro',
        answerKey: 'faqTransactionsIntroAnswer',
      },
      {
        id: 'transactions-history',
        questionKey: 'faqTransactionsHistory',
        answerKey: 'faqTransactionsHistoryAnswer',
      },
    ],
  },
  {
    id: 'data',
    titleKey: 'faqCategoryData',
    icon: Database,
    items: [
      {
        id: 'data-intro',
        questionKey: 'faqDataIntro',
        answerKey: 'faqDataIntroAnswer',
      },
      {
        id: 'data-sources',
        questionKey: 'faqDataSources',
        answerKey: 'faqDataSourcesAnswer',
      },
    ],
  },
  {
    id: 'strategy-market',
    titleKey: 'faqCategoryStrategyMarket',
    icon: Store,
    items: [
      {
        id: 'strategy-market-intro',
        questionKey: 'faqStrategyMarketIntro',
        answerKey: 'faqStrategyMarketIntroAnswer',
      },
      {
        id: 'strategy-market-share',
        questionKey: 'faqStrategyMarketShare',
        answerKey: 'faqStrategyMarketShareAnswer',
      },
    ],
  },
  {
    id: 'chaos-studio',
    titleKey: 'faqCategoryChaosStudio',
    icon: TrendingUp,
    items: [
      {
        id: 'chaos-studio-intro',
        questionKey: 'faqChaosStudioIntro',
        answerKey: 'faqChaosStudioIntroAnswer',
      },
      {
        id: 'chaos-studio-features',
        questionKey: 'faqChaosStudioFeatures',
        answerKey: 'faqChaosStudioFeaturesAnswer',
      },
    ],
  },
  {
    id: 'chaos',
    titleKey: 'faqCategoryChaos',
    icon: TrendingUp,
    items: [
      {
        id: 'chaos-intro',
        questionKey: 'faqChaosIntro',
        answerKey: 'faqChaosIntroAnswer',
      },
      {
        id: 'chaos-config',
        questionKey: 'faqChaosConfig',
        answerKey: 'faqChaosConfigAnswer',
      },
    ],
  },
]
