import { useState, useEffect, useCallback, useRef } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import {
  Plus,
  Copy,
  Trash2,
  Check,
  ChevronDown,
  ChevronRight,
  BarChart3,
  Target,
  Shield,
  Zap,
  Activity,
  Save,
  Sparkles,
  Eye,
  Play,
  FileText,
  Loader2,
  RefreshCw,
  Clock,
  Bot,
  Terminal,
  Code,
  Send,
  Download,
  Upload,
  Globe,
  Dna,
} from 'lucide-react'
import type { Strategy, StrategyConfig, AIModel } from '../types'
import { confirmToast, notify } from '../lib/notify'
import { CoinSourceEditor } from '../components/strategy/CoinSourceEditor'
import { IndicatorEditor } from '../components/strategy/IndicatorEditor'
import { RiskControlEditor } from '../components/strategy/RiskControlEditor'
import { PromptSectionsEditor } from '../components/strategy/PromptSectionsEditor'
import { PublishSettingsEditor } from '../components/strategy/PublishSettingsEditor'
import { DeepVoidBackground } from '../components/DeepVoidBackground'

const API_BASE = import.meta.env.VITE_API_BASE || ''

export function ChaosStudioPage() {
  const { token } = useAuth()
  const { language } = useLanguage()

  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [selectedStrategy, setSelectedStrategy] = useState<Strategy | null>(
    null
  )
  const [editingConfig, setEditingConfig] = useState<StrategyConfig | null>(
    null
  )
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [hasChanges, setHasChanges] = useState(false)

  // AI Models for test run
  const [aiModels, setAiModels] = useState<AIModel[]>([])
  const [selectedModelId, setSelectedModelId] = useState<string>('')

  // Sidebar resizable state
  const [sidebarWidth, setSidebarWidth] = useState(260)
  const isResizing = useRef(false)

  const startResizing = useCallback(() => {
    isResizing.current = true
    document.addEventListener('mousemove', resize)
    document.addEventListener('mouseup', stopResizing)
    document.body.style.userSelect = 'none'
  }, [])

  const stopResizing = useCallback(() => {
    isResizing.current = false
    document.removeEventListener('mousemove', resize)
    document.removeEventListener('mouseup', stopResizing)
    document.body.style.userSelect = ''
  }, [])

  const resize = useCallback((e: MouseEvent) => {
    if (isResizing.current) {
      const newWidth = Math.max(200, Math.min(600, e.clientX))
      setSidebarWidth(newWidth)
    }
  }, [])

  // Accordion states for left panel
  const [expandedSections, setExpandedSections] = useState({
    coinSource: false,
    indicators: false,
    riskControl: false,
    promptSections: false,
    customPrompt: true, // Default open for Chaos
    publishSettings: false,
  })

  // Right panel states
  const [activeRightTab, setActiveRightTab] = useState<'prompt' | 'test'>(
    'prompt'
  )
  const [promptPreview, setPromptPreview] = useState<{
    system_prompt: string
    user_prompt?: string
    prompt_variant: string
    config_summary: Record<string, unknown>
  } | null>(null)
  const [isLoadingPrompt, setIsLoadingPrompt] = useState(false)
  const [selectedVariant, setSelectedVariant] = useState('s1') // Default Chaos variant

  // AI Test Run states
  const [aiTestResult, setAiTestResult] = useState<{
    system_prompt?: string
    user_prompt?: string
    ai_response?: string
    reasoning?: string
    decisions?: unknown[]
    error?: string
    duration_ms?: number
  } | null>(null)
  const [isRunningAiTest, setIsRunningAiTest] = useState(false)

  const toggleSection = (section: keyof typeof expandedSections) => {
    setExpandedSections((prev) => ({
      ...prev,
      [section]: !prev[section],
    }))
  }

  // Fetch AI Models
  const fetchAiModels = useCallback(async () => {
    if (!token) return
    try {
      const response = await fetch(`${API_BASE}/api/models`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        const allModels = Array.isArray(data) ? data : data.models || []
        const enabledModels = allModels.filter((m: AIModel) => m.enabled)
        setAiModels(enabledModels)
        if (enabledModels.length > 0 && !selectedModelId) {
          setSelectedModelId(enabledModels[0].id)
        }
      }
    } catch (err) {
      console.error('Failed to fetch AI models:', err)
    }
  }, [token, selectedModelId])

  // Helper to check if a strategy is Chaos type
  const isChaosStrategy = (strategy: Strategy): boolean => {
    try {
      // Check if custom_prompt contains the prompt_meta type: chaos
      if (strategy.config?.custom_prompt) {
        if (strategy.config.custom_prompt.includes('"type": "chaos"') || 
            strategy.config.custom_prompt.includes('"type":"chaos"')) {
          return true;
        }
        // Fallback for legacy check
        if (strategy.config.custom_prompt.includes('Chaos Trader')) {
            return true;
        }
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  // Fetch strategies - FILTERED for Chaos
  const fetchStrategies = useCallback(async () => {
    if (!token) return
    try {
      const response = await fetch(`${API_BASE}/api/strategies`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!response.ok) throw new Error('Failed to fetch strategies')
      const data = await response.json()
      
      // Filter only Chaos strategies
      const chaosStrategies = (data.strategies || []).filter(isChaosStrategy)
      setStrategies(chaosStrategies)

      // Select active or first strategy
      const active = chaosStrategies.find((s: Strategy) => s.is_active)
      if (active) {
        setSelectedStrategy(active)
        setEditingConfig(active.config)
        // Set variant from config if available, otherwise default to s1
        if (active.config?.prompt_variant) {
            setSelectedVariant(active.config.prompt_variant)
        } else {
            setSelectedVariant('s1')
        }
      } else if (chaosStrategies.length > 0) {
        setSelectedStrategy(chaosStrategies[0])
        setEditingConfig(chaosStrategies[0].config)
        // Set variant from config if available, otherwise default to s1
        if (chaosStrategies[0].config?.prompt_variant) {
            setSelectedVariant(chaosStrategies[0].config.prompt_variant)
        } else {
            setSelectedVariant('s1')
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsLoading(false)
    }
  }, [token])

  useEffect(() => {
    fetchStrategies()
    fetchAiModels()
  }, [fetchStrategies, fetchAiModels])

  // Track previous language to detect actual changes
  const prevLanguageRef = useRef(language)

  // When language changes, update prompt sections to match the new language
  useEffect(() => {
    const updatePromptSectionsForLanguage = async () => {
      // Only update if language actually changed (not on initial mount)
      if (prevLanguageRef.current === language) return
      prevLanguageRef.current = language

      if (!token) return

      try {
        // Fetch default config for the new language
        const response = await fetch(
          `${API_BASE}/api/strategies/default-config?lang=${language}`,
          { headers: { Authorization: `Bearer ${token}` } }
        )
        if (!response.ok) return
        const defaultConfig = await response.json()

        // Update only the prompt sections and language field
        setEditingConfig((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            language: language as 'zh' | 'en',
            prompt_sections: defaultConfig.prompt_sections,
          }
        })
        setHasChanges(true)
      } catch (err) {
        console.error('Failed to update prompt sections for language:', err)
      }
    }

    updatePromptSectionsForLanguage()
  }, [language, token]) // Only trigger when language changes

  // Create new Chaos strategy
  const handleCreateStrategy = async () => {
    if (!token) return
    try {
      const configResponse = await fetch(
        `${API_BASE}/api/strategies/default-config?lang=${language}`,
        { headers: { Authorization: `Bearer ${token}` } }
      )
      const defaultConfig = await configResponse.json()
      
      // Inject Chaos template into custom_prompt
      const chaosTemplate = JSON.stringify({
        prompt_meta: {
            type: "chaos",
            prompt_name: language === 'zh' ? '新 Chaos 策略' : 'New Chaos Strategy',
            author: "User",
            version: "1.0"
        },
        risk_r_config: {
            mode: "standard",
            min_risk_r: 1.5
        }
      }, null, 2);

      const chaosConfig = {
          ...defaultConfig,
          custom_prompt: chaosTemplate
      };

      const response = await fetch(`${API_BASE}/api/strategies`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: language === 'zh' ? '新 Chaos 策略' : 'New Chaos Strategy',
          description: 'Created in Chaos Studio',
          config: chaosConfig,
        }),
      })
      if (!response.ok) throw new Error('Failed to create strategy')
      const result = await response.json()
      await fetchStrategies()
      
      // Auto-select the newly created strategy
      if (result.id) {
        const now = new Date().toISOString()
        const newStrategy = {
          id: result.id,
          name: language === 'zh' ? '新 Chaos 策略' : 'New Chaos Strategy',
          description: 'Created in Chaos Studio',
          is_active: false,
          is_default: false,
          is_public: false,
          config_visible: true,
          config: chaosConfig,
          created_at: now,
          updated_at: now,
        }
        setSelectedStrategy(newStrategy)
        setEditingConfig(chaosConfig)
        setHasChanges(false)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  // Delete strategy
  const handleDeleteStrategy = async (id: string) => {
    if (!token) return

    const confirmed = await confirmToast(
      language === 'zh' ? '确定删除此策略？' : 'Delete this strategy?',
      {
        title: language === 'zh' ? '确认删除' : 'Confirm Delete',
        okText: language === 'zh' ? '删除' : 'Delete',
        cancelText: language === 'zh' ? '取消' : 'Cancel',
      }
    )
    if (!confirmed) return

    try {
      const response = await fetch(`${API_BASE}/api/strategies/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!response.ok) throw new Error('Failed to delete strategy')
      notify.success(language === 'zh' ? '策略已删除' : 'Strategy deleted')
      // Clear selection if deleted strategy was selected
      if (selectedStrategy?.id === id) {
        setSelectedStrategy(null)
        setEditingConfig(null)
        setHasChanges(false)
      }
      await fetchStrategies()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Unknown error'
      setError(errorMsg)
      notify.error(errorMsg)
    }
  }

  // Duplicate strategy
  const handleDuplicateStrategy = async (id: string) => {
    if (!token) return
    try {
      const response = await fetch(
        `${API_BASE}/api/strategies/${id}/duplicate`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: language === 'zh' ? '策略副本' : 'Strategy Copy',
          }),
        }
      )
      if (!response.ok) throw new Error('Failed to duplicate strategy')
      await fetchStrategies()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  // Activate strategy
  const handleActivateStrategy = async (id: string) => {
    if (!token) return
    try {
      const response = await fetch(
        `${API_BASE}/api/strategies/${id}/activate`,
        {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
        }
      )
      if (!response.ok) throw new Error('Failed to activate strategy')
      await fetchStrategies()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  // Export strategy as JSON file
  const handleExportStrategy = (strategy: Strategy) => {
    const exportData = {
      name: strategy.name,
      description: strategy.description,
      config: strategy.config,
      exported_at: new Date().toISOString(),
      version: '1.0',
    }
    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chaos_strategy_${strategy.name.replace(/\s+/g, '_')}_${new Date().toISOString().split('T')[0]}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    notify.success(language === 'zh' ? '策略已导出' : 'Strategy exported')
  }

  // Import strategy from JSON file
  const handleImportStrategy = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0]
    if (!file || !token) return

    try {
      const text = await file.text()
      const importData = JSON.parse(text)

      // Validate imported data
      if (!importData.config || !importData.name) {
        throw new Error(
          language === 'zh' ? '无效的策略文件' : 'Invalid strategy file'
        )
      }

      // Check if it's a Chaos strategy (optional, but good for UX)
      // We allow importing any strategy here, but maybe warn if not chaos?
      // For now, just import.

      // Create new strategy with imported config
      const response = await fetch(`${API_BASE}/api/strategies`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: `${importData.name} (${language === 'zh' ? '导入' : 'Imported'})`,
          description: importData.description || '',
          config: importData.config,
        }),
      })
      if (!response.ok) throw new Error('Failed to import strategy')

      notify.success(language === 'zh' ? '策略已导入' : 'Strategy imported')
      await fetchStrategies()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Unknown error'
      notify.error(errorMsg)
    } finally {
      // Reset file input
      event.target.value = ''
    }
  }

  // Save strategy
  const handleSaveStrategy = async () => {
    if (!token || !selectedStrategy || !editingConfig) return
    setIsSaving(true)
    try {
      // Always sync the config language with the current interface language
      const configWithLanguage = {
        ...editingConfig,
        language: language as 'zh' | 'en',
        prompt_variant: selectedVariant, // Also save the variant preference if backend supports
      }
      const response = await fetch(
        `${API_BASE}/api/strategies/${selectedStrategy.id}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: selectedStrategy.name,
            description: selectedStrategy.description,
            config: configWithLanguage,
            is_public: selectedStrategy.is_public,
            config_visible: selectedStrategy.config_visible,
          }),
        }
      )
      if (!response.ok) throw new Error('Failed to save strategy')
      setHasChanges(false)
      notify.success(language === 'zh' ? '策略已保存' : 'Strategy saved')
      await fetchStrategies()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsSaving(false)
    }
  }

  // Update config section
  const updateConfig = <K extends keyof StrategyConfig>(
    section: K,
    value: StrategyConfig[K]
  ) => {
    if (!editingConfig) return
    setEditingConfig({
      ...editingConfig,
      [section]: value,
    })
    setHasChanges(true)
  }

  // Fetch prompt preview
  const fetchPromptPreview = async () => {
    if (!token || !editingConfig) return
    setIsLoadingPrompt(true)
    try {
      const response = await fetch(
        `${API_BASE}/api/strategies/preview-prompt`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            config: editingConfig,
            account_equity: 1000,
            prompt_variant: selectedVariant,
          }),
        }
      )
      if (!response.ok) throw new Error('Failed to fetch prompt preview')
      const data = await response.json()
      setPromptPreview(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsLoadingPrompt(false)
    }
  }

  // Run AI test with real AI model
  const runAiTest = async () => {
    if (!token || !editingConfig || !selectedModelId) return
    setIsRunningAiTest(true)
    setAiTestResult(null)
    try {
      const response = await fetch(`${API_BASE}/api/strategies/test-run`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          config: editingConfig,
          prompt_variant: selectedVariant,
          ai_model_id: selectedModelId,
          run_real_ai: true,
        }),
      })
      if (!response.ok) throw new Error('Failed to run AI test')
      const data = await response.json()
      setAiTestResult(data)
    } catch (err) {
      setAiTestResult({
        error: err instanceof Error ? err.message : 'Unknown error',
      })
    } finally {
      setIsRunningAiTest(false)
    }
  }

  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      chaosStudio: { zh: 'Chaos 实验室', en: 'Chaos Studio' },
      subtitle: {
        zh: '高级策略与对抗性测试环境',
        en: 'Advanced strategies & adversarial testing',
      },
      strategies: { zh: 'Chaos 策略', en: 'Chaos Strategies' },
      newStrategy: { zh: '新建 Chaos', en: 'New Chaos' },
      coinSource: { zh: '币种来源', en: 'Coin Source' },
      indicators: { zh: '技术指标', en: 'Indicators' },
      riskControl: { zh: '风控参数', en: 'Risk Control' },
      promptSections: { zh: 'Prompt 编辑', en: 'Prompt Editor' },
      customPrompt: { zh: 'Chaos 核心配置 (JSON)', en: 'Chaos Core Config (JSON)' },
      save: { zh: '保存', en: 'Save' },
      saving: { zh: '保存中...', en: 'Saving...' },
      activate: { zh: '激活', en: 'Activate' },
      active: { zh: '激活中', en: 'Active' },
      default: { zh: '默认', en: 'Default' },
      promptPreview: { zh: 'Prompt 预览', en: 'Prompt Preview' },
      aiTestRun: { zh: 'Chaos 对抗测试', en: 'Chaos Test Run' },
      systemPrompt: { zh: 'System Prompt', en: 'System Prompt' },
      userPrompt: { zh: 'User Prompt', en: 'User Prompt' },
      loadPrompt: { zh: '生成 Prompt', en: 'Generate Prompt' },
      refreshPrompt: { zh: '刷新', en: 'Refresh' },
      promptVariant: { zh: 'Chaos 变体', en: 'Chaos Variant' },
      balanced: { zh: '平衡', en: 'Balanced' },
      aggressive: { zh: '激进', en: 'Aggressive' },
      conservative: { zh: '保守', en: 'Conservative' },
      none: { zh: '无 (自定义)', en: 'None (Custom)' },
      s1: { zh: 'S1 (主力/基线)', en: 'S1 (SWING_CORE)' },
      t1: { zh: 'T1 (慢趋势)', en: 'T1 (TREND_FOLLOW_SLOW)' },
      d1: { zh: 'D1 (日内波段)', en: 'D1 (INTRADAY_SWING)' },
      r1: { zh: 'R1 (震荡防御)', en: 'R1 (RANGE_DEFENSIVE)' },
      x1: { zh: 'X1 (实验/微结构)', en: 'X1 (SCALP_EXPERIMENT)' },
      selectModel: { zh: '选择 AI 模型', en: 'Select AI Model' },
      runTest: { zh: '运行测试', en: 'Run Test' },
      running: { zh: '运行中...', en: 'Running...' },
      aiOutput: { zh: 'AI 输出', en: 'AI Output' },
      reasoning: { zh: '思维链', en: 'Reasoning' },
      decisions: { zh: '决策', en: 'Decisions' },
      duration: { zh: '耗时', en: 'Duration' },
      noModel: {
        zh: '请先配置 AI 模型',
        en: 'Please configure AI model first',
      },
      testNote: {
        zh: '使用真实 AI 模型测试，不执行交易',
        en: 'Test with real AI, no trading',
      },
      publishSettings: { zh: '发布设置', en: 'Publish' },
    }
    return translations[key]?.[language] || key
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[70vh]">
        <div className="text-center">
          <div className="relative">
            <div className="w-16 h-16 rounded-full border-4 border-yellow-500/20 border-t-yellow-500 animate-spin" />
            <Zap className="w-6 h-6 text-yellow-500 absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2" />
          </div>
        </div>
      </div>
    )
  }

  const configSections = [
    // Highlighted Chaos Config Section
    {
      key: 'customPrompt' as const,
      icon: Dna,
      color: '#a855f7',
      title: t('customPrompt'),
      content: editingConfig && (
        <div>
          <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {language === 'zh'
              ? 'Chaos 策略的核心配置文件，必须包含 prompt_meta: { type: "chaos" }'
              : 'Core configuration for Chaos strategy. Must include prompt_meta: { type: "chaos" }'}
          </p>
          <div className="mb-2 flex items-center gap-2">
            <select
                value={selectedVariant}
                onChange={(e) => {
                  setSelectedVariant(e.target.value)
                  setHasChanges(true)
                }}
                className="px-2 py-1.5 rounded text-xs bg-nofx-bg border border-nofx-gold/20 text-nofx-text"
            >
                <option value="none">{t('none')}</option>
                <option value="s1">{t('s1')}</option>
                <option value="t1">{t('t1')}</option>
                <option value="d1">{t('d1')}</option>
                <option value="r1">{t('r1')}</option>
                <option value="x1">{t('x1')}</option>
            </select>
            <span className="text-xs text-nofx-text-muted">← {t('promptVariant')}</span>
          </div>
          <textarea
            value={editingConfig.custom_prompt || ''}
            onChange={(e) => updateConfig('custom_prompt', e.target.value)}
            disabled={selectedStrategy?.is_default}
            placeholder={
              language === 'zh'
                ? '输入 Chaos JSON 配置...'
                : 'Enter Chaos JSON config...'
            }
            className="w-full h-64 px-3 py-2 rounded-lg resize-none font-mono text-xs"
            style={{
              background: '#0B0E11',
              border: '1px solid #a855f7',
              color: '#EAECEF',
            }}
          />
        </div>
      ),
    },
    // Standard sections collapsed by default
    {
      key: 'coinSource' as const,
      icon: Target,
      color: '#F0B90B',
      title: t('coinSource'),
      content: editingConfig && (
        <CoinSourceEditor
          config={editingConfig.coin_source}
          onChange={(coinSource) => updateConfig('coin_source', coinSource)}
          disabled={selectedStrategy?.is_default}
          language={language}
        />
      ),
    },
    {
      key: 'indicators' as const,
      icon: BarChart3,
      color: '#0ECB81',
      title: t('indicators'),
      content: editingConfig && (
        <IndicatorEditor
          config={editingConfig.indicators}
          onChange={(indicators) => updateConfig('indicators', indicators)}
          disabled={selectedStrategy?.is_default}
          language={language}
        />
      ),
    },
    {
      key: 'riskControl' as const,
      icon: Shield,
      color: '#F6465D',
      title: t('riskControl'),
      content: editingConfig && (
        <RiskControlEditor
          config={editingConfig.risk_control}
          onChange={(riskControl) => updateConfig('risk_control', riskControl)}
          disabled={selectedStrategy?.is_default}
          language={language}
        />
      ),
    },
    {
      key: 'promptSections' as const,
      icon: FileText,
      color: '#60a5fa',
      title: t('promptSections'),
      content: editingConfig && (
        <PromptSectionsEditor
          config={editingConfig.prompt_sections}
          onChange={(promptSections) =>
            updateConfig('prompt_sections', promptSections)
          }
          disabled={selectedStrategy?.is_default}
          language={language}
        />
      ),
    },
    {
      key: 'publishSettings' as const,
      icon: Globe,
      color: '#0ECB81',
      title: t('publishSettings'),
      content: selectedStrategy && (
        <PublishSettingsEditor
          isPublic={selectedStrategy.is_public ?? false}
          configVisible={selectedStrategy.config_visible ?? true}
          onIsPublicChange={(value) => {
            setSelectedStrategy({ ...selectedStrategy, is_public: value })
            setHasChanges(true)
          }}
          onConfigVisibleChange={(value) => {
            setSelectedStrategy({ ...selectedStrategy, config_visible: value })
            setHasChanges(true)
          }}
          disabled={selectedStrategy?.is_default}
          language={language}
        />
      ),
    },
  ]

  return (
    <DeepVoidBackground className="h-[calc(100vh-64px)] flex flex-col bg-nofx-bg relative overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-nofx-gold/20 bg-nofx-bg/60 backdrop-blur-md z-10">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-gradient-to-br from-purple-600 to-indigo-600">
              <Dna className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-nofx-text">
                {t('chaosStudio')}
              </h1>
              <p className="text-xs text-nofx-text-muted">{t('subtitle')}</p>
            </div>
          </div>
          {error && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs bg-nofx-danger/10 text-nofx-danger">
              {error}
              <button
                onClick={() => setError(null)}
                className="hover:underline"
              >
                ×
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Main Content - Three Columns */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Column - Strategy List */}
        <div
          style={{ width: sidebarWidth }}
          className="flex-shrink-0 border-r border-nofx-gold/20 overflow-y-auto bg-nofx-bg/30 backdrop-blur-sm z-10"
        >
          <div className="p-2">
            <div className="flex items-center justify-between mb-2 px-2">
              <span className="text-xs font-medium text-nofx-text-muted">
                {t('strategies')}
              </span>
              <div className="flex items-center gap-1">
                {/* Import button with hidden file input */}
                <label
                  className="p-1 rounded hover:bg-white/10 transition-colors cursor-pointer text-nofx-text-muted hover:text-white"
                  title={language === 'zh' ? '导入策略' : 'Import Strategy'}
                >
                  <Upload className="w-4 h-4" />
                  <input
                    type="file"
                    accept=".json"
                    onChange={handleImportStrategy}
                    className="hidden"
                  />
                </label>
                <button
                  onClick={handleCreateStrategy}
                  className="p-1 rounded hover:bg-white/10 transition-colors text-purple-400"
                  title={language === 'zh' ? '新建 Chaos 策略' : 'New Chaos Strategy'}
                >
                  <Plus className="w-4 h-4" />
                </button>
              </div>
            </div>
            <div className="space-y-1">
              {strategies.map((strategy) => (
                <div
                  key={strategy.id}
                  onClick={() => {
                    setSelectedStrategy(strategy)
                    setEditingConfig(strategy.config)
                    // Update variant state when switching strategy
                    if (strategy.config?.prompt_variant) {
                        setSelectedVariant(strategy.config.prompt_variant)
                    } else {
                        setSelectedVariant('s1')
                    }
                    setHasChanges(false)
                    setPromptPreview(null)
                    setAiTestResult(null)
                  }}
                  className={`group px-2 py-2 rounded-lg cursor-pointer transition-all ${
                    selectedStrategy?.id === strategy.id
                      ? 'ring-1 ring-purple-500/50 bg-purple-500/10 shadow-[0_0_15px_rgba(168,85,247,0.1)]'
                      : 'hover:bg-nofx-bg-lighter/60 hover:ring-1 hover:ring-purple-500/20 bg-transparent'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm truncate text-nofx-text">
                      {strategy.name}
                    </span>
                    <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleExportStrategy(strategy)
                        }}
                        className="p-1 rounded hover:bg-white/10 text-nofx-text-muted hover:text-white"
                        title={language === 'zh' ? '导出' : 'Export'}
                      >
                        <Download className="w-3 h-3" />
                      </button>
                      {!strategy.is_default && (
                        <>
                          <button
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDuplicateStrategy(strategy.id)
                            }}
                            className="p-1 rounded hover:bg-white/10 text-nofx-text-muted hover:text-white"
                            title={language === 'zh' ? '复制' : 'Duplicate'}
                          >
                            <Copy className="w-3 h-3" />
                          </button>
                          <button
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDeleteStrategy(strategy.id)
                            }}
                            className="p-1 rounded hover:bg-nofx-danger/20 text-nofx-danger"
                            title={language === 'zh' ? '删除' : 'Delete'}
                          >
                            <Trash2 className="w-3 h-3" />
                          </button>
                        </>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-1 mt-1 flex-wrap">
                    {strategy.is_active && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-nofx-success/15 text-nofx-success">
                        {t('active')}
                      </span>
                    )}
                    {strategy.is_default && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-nofx-gold/15 text-nofx-gold">
                        {t('default')}
                      </span>
                    )}
                    {strategy.is_public && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded flex items-center gap-0.5 bg-blue-400/15 text-blue-400">
                        <Globe className="w-2.5 h-2.5" />
                        {language === 'zh' ? '公开' : 'Public'}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Resizer */}
        <div
          className="w-1 cursor-col-resize hover:bg-purple-500/50 transition-colors active:bg-purple-500 z-20 flex-shrink-0"
          onMouseDown={startResizing}
        />

        {/* Middle Column - Config Editor */}
        <div className="flex-1 min-w-0 overflow-y-auto border-r border-nofx-gold/20">
          {selectedStrategy && editingConfig ? (
            <div className="p-4">
              {/* Strategy Name & Actions */}
              <div className="flex items-center justify-between mb-4">
                <div className="flex-1 min-w-0">
                  <input
                    type="text"
                    value={selectedStrategy.name}
                    onChange={(e) => {
                      setSelectedStrategy({
                        ...selectedStrategy,
                        name: e.target.value,
                      })
                      setHasChanges(true)
                    }}
                    disabled={selectedStrategy.is_default}
                    className="text-lg font-bold bg-transparent border-none outline-none w-full text-nofx-text placeholder-nofx-text-muted"
                  />
                  <input
                    type="text"
                    value={selectedStrategy.description || ''}
                    onChange={(e) => {
                      setSelectedStrategy({
                        ...selectedStrategy,
                        description: e.target.value,
                      })
                      setHasChanges(true)
                    }}
                    disabled={selectedStrategy.is_default}
                    placeholder={
                      language === 'zh'
                        ? '添加策略简介...'
                        : 'Add strategy description...'
                    }
                    className="text-xs bg-transparent border-none outline-none w-full text-nofx-text-muted placeholder-nofx-text-muted/50 mt-1"
                  />
                  {hasChanges && (
                    <span className="text-xs text-nofx-gold">
                      ● {language === 'zh' ? '未保存' : 'Unsaved'}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {!selectedStrategy.is_active && (
                    <button
                      onClick={() =>
                        handleActivateStrategy(selectedStrategy.id)
                      }
                      className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs transition-colors bg-nofx-success/10 border border-nofx-success/30 text-nofx-success hover:bg-nofx-success/20"
                    >
                      <Check className="w-3 h-3" />
                      {t('activate')}
                    </button>
                  )}
                  {!selectedStrategy.is_default && (
                    <button
                      onClick={handleSaveStrategy}
                      disabled={isSaving || !hasChanges}
                      className={`flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors disabled:opacity-50
                        ${hasChanges ? 'bg-nofx-gold text-black hover:bg-yellow-500' : 'bg-nofx-bg-lighter text-nofx-text-muted cursor-not-allowed'}`}
                    >
                      <Save className="w-3 h-3" />
                      {isSaving ? t('saving') : t('save')}
                    </button>
                  )}
                </div>
              </div>

              {/* Config Sections */}
              <div className="space-y-2">
                {configSections.map(
                  ({ key, icon: Icon, color, title, content }) => (
                    <div
                      key={key}
                      className={`rounded-lg overflow-hidden bg-nofx-bg-lighter border ${key === 'customPrompt' ? 'border-purple-500/40' : 'border-nofx-gold/20'}`}
                    >
                      <button
                        onClick={() => toggleSection(key)}
                        className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-white/5 transition-colors"
                      >
                        <div className="flex items-center gap-2">
                          <Icon className="w-4 h-4" style={{ color }} />
                          <span className="text-sm font-medium text-nofx-text">
                            {title}
                          </span>
                        </div>
                        {expandedSections[key] ? (
                          <ChevronDown className="w-4 h-4 text-nofx-text-muted" />
                        ) : (
                          <ChevronRight className="w-4 h-4 text-nofx-text-muted" />
                        )}
                      </button>
                      {expandedSections[key] && (
                        <div className="px-3 pb-3">{content}</div>
                      )}
                    </div>
                  )
                )}
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <Dna className="w-12 h-12 mx-auto mb-2 opacity-30 text-purple-500" />
                <p className="text-sm text-nofx-text-muted">
                  {language === 'zh'
                    ? '选择或创建 Chaos 策略'
                    : 'Select or create a Chaos strategy'}
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Right Column - Prompt Preview & AI Test */}
        <div className="w-[420px] flex-shrink-0 flex flex-col overflow-hidden">
          {/* Tabs */}
          <div className="flex-shrink-0 flex border-b border-nofx-gold/20">
            <button
              onClick={() => setActiveRightTab('prompt')}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors ${
                activeRightTab === 'prompt'
                  ? 'border-b-2 border-purple-500 text-purple-500'
                  : 'opacity-60 hover:opacity-100 text-nofx-text-muted'
              }`}
            >
              <Eye className="w-4 h-4" />
              {t('promptPreview')}
            </button>
            <button
              onClick={() => setActiveRightTab('test')}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors ${
                activeRightTab === 'test'
                  ? 'border-b-2 border-green-500 text-green-500'
                  : 'opacity-60 hover:opacity-100 text-nofx-text-muted'
              }`}
            >
              <Play className="w-4 h-4" />
              {t('aiTestRun')}
            </button>
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-y-auto">
            {activeRightTab === 'prompt' ? (
              /* Prompt Preview Tab */
              <div className="p-3 space-y-3">
                {/* Controls */}
                <div className="flex items-center gap-2 flex-wrap">
                  <button
                    onClick={fetchPromptPreview}
                    disabled={isLoadingPrompt || !editingConfig}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors disabled:opacity-50 bg-purple-600 hover:bg-purple-700 text-white w-full justify-center"
                  >
                    {isLoadingPrompt ? (
                      <Loader2 className="w-3 h-3 animate-spin" />
                    ) : (
                      <RefreshCw className="w-3 h-3" />
                    )}
                    {promptPreview ? t('refreshPrompt') : t('loadPrompt')}
                  </button>
                </div>

                {promptPreview ? (
                  <>
                    {/* Config Summary */}
                    <div className="p-2 rounded-lg bg-nofx-bg border border-nofx-gold/20">
                      <div className="flex items-center gap-1.5 mb-2">
                        <Code className="w-3 h-3 text-purple-500" />
                        <span className="text-xs font-medium text-purple-500">
                          Config
                        </span>
                      </div>
                      <div className="grid grid-cols-3 gap-2 text-xs">
                        {Object.entries(promptPreview.config_summary || {}).map(
                          ([key, value]) => (
                            <div key={key}>
                              <div className="text-nofx-text-muted">
                                {key.replace(/_/g, ' ')}
                              </div>
                              <div className="text-nofx-text">
                                {String(value)}
                              </div>
                            </div>
                          )
                        )}
                      </div>
                    </div>

                    {/* System Prompt */}
                    <div>
                      <div className="flex items-center justify-between mb-1.5">
                        <div className="flex items-center gap-1.5">
                          <FileText className="w-3 h-3 text-purple-500" />
                          <span className="text-xs font-medium text-nofx-text">
                            {t('systemPrompt')}
                          </span>
                        </div>
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-nofx-bg-lighter text-nofx-text-muted">
                          {promptPreview.system_prompt.length.toLocaleString()}{' '}
                          chars
                        </span>
                      </div>
                      <pre
                        className="p-2 rounded-lg text-[11px] font-mono overflow-auto bg-nofx-bg border border-nofx-gold/20 text-nofx-text"
                        style={{ maxHeight: '400px' }}
                      >
                        {promptPreview.system_prompt}
                      </pre>
                    </div>
                  </>
                ) : (
                  <div className="flex flex-col items-center justify-center py-12 text-nofx-text-muted">
                    <Eye className="w-10 h-10 mb-2 opacity-30" />
                    <p className="text-sm">
                      {language === 'zh'
                        ? '点击生成 Prompt 预览'
                        : 'Click to generate prompt preview'}
                    </p>
                  </div>
                )}
              </div>
            ) : (
              /* AI Test Tab */
              <div className="p-3 space-y-3">
                {/* Controls */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Bot className="w-4 h-4 text-green-500" />
                    <span className="text-xs font-medium text-nofx-text">
                      {t('selectModel')}
                    </span>
                  </div>
                  {aiModels.length > 0 ? (
                    <select
                      value={selectedModelId}
                      onChange={(e) => setSelectedModelId(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg text-sm bg-nofx-bg border border-nofx-gold/20 text-nofx-text"
                    >
                      {aiModels.map((model) => (
                        <option key={model.id} value={model.id}>
                          {model.name} ({model.provider})
                        </option>
                      ))}
                    </select>
                  ) : (
                    <div className="px-3 py-2 rounded-lg text-sm bg-nofx-danger/10 text-nofx-danger">
                      {t('noModel')}
                    </div>
                  )}

                  <div className="flex items-center gap-2">
                    <button
                      onClick={runAiTest}
                      disabled={
                        isRunningAiTest || !editingConfig || !selectedModelId
                      }
                      className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all disabled:opacity-50 text-white shadow-lg shadow-green-500/20 bg-gradient-to-br from-green-500 to-green-600"
                    >
                      {isRunningAiTest ? (
                        <>
                          <Loader2 className="w-4 h-4 animate-spin" />
                          {t('running')}
                        </>
                      ) : (
                        <>
                          <Send className="w-4 h-4" />
                          {t('runTest')}
                        </>
                      )}
                    </button>
                  </div>
                  <p className="text-[10px] text-nofx-text-muted">
                    {t('testNote')}
                  </p>
                </div>

                {/* Test Results */}
                {aiTestResult ? (
                  <div className="space-y-3">
                    {aiTestResult.error ? (
                      <div className="p-3 rounded-lg bg-nofx-danger/10 border border-nofx-danger/30">
                        <p className="text-sm text-nofx-danger">
                          {aiTestResult.error}
                        </p>
                      </div>
                    ) : (
                      <>
                        {aiTestResult.duration_ms && (
                          <div className="flex items-center gap-2">
                            <Clock className="w-3 h-3 text-nofx-text-muted" />
                            <span className="text-xs text-nofx-text-muted">
                              {t('duration')}:{' '}
                              {(aiTestResult.duration_ms / 1000).toFixed(2)}s
                            </span>
                          </div>
                        )}

                        {/* User Prompt Input */}
                        {aiTestResult.user_prompt && (
                          <div>
                            <div className="flex items-center gap-1.5 mb-1.5">
                              <Terminal className="w-3 h-3 text-blue-400" />
                              <span className="text-xs font-medium text-nofx-text">
                                {t('userPrompt')} (Input)
                              </span>
                            </div>
                            <pre
                              className="p-2 rounded-lg text-[10px] font-mono overflow-auto bg-nofx-bg border border-nofx-gold/20 text-nofx-text"
                              style={{ maxHeight: '200px' }}
                            >
                              {aiTestResult.user_prompt}
                            </pre>
                          </div>
                        )}

                        {/* AI Reasoning */}
                        {aiTestResult.reasoning && (
                          <div>
                            <div className="flex items-center gap-1.5 mb-1.5">
                              <Sparkles className="w-3 h-3 text-nofx-gold" />
                              <span className="text-xs font-medium text-nofx-text">
                                {t('reasoning')}
                              </span>
                            </div>
                            <pre
                              className="p-2 rounded-lg text-[10px] font-mono overflow-auto whitespace-pre-wrap bg-nofx-bg border border-nofx-gold/30 text-nofx-text"
                              style={{ maxHeight: '200px' }}
                            >
                              {aiTestResult.reasoning}
                            </pre>
                          </div>
                        )}

                        {/* AI Decisions */}
                        {aiTestResult.decisions &&
                          aiTestResult.decisions.length > 0 && (
                            <div>
                              <div className="flex items-center gap-1.5 mb-1.5">
                                <Activity className="w-3 h-3 text-green-500" />
                                <span className="text-xs font-medium text-nofx-text">
                                  {t('decisions')}
                                </span>
                              </div>
                              <pre
                                className="p-2 rounded-lg text-[10px] font-mono overflow-auto bg-nofx-bg border border-green-500/30 text-nofx-text"
                                style={{ maxHeight: '200px' }}
                              >
                                {JSON.stringify(
                                  aiTestResult.decisions,
                                  null,
                                  2
                                )}
                              </pre>
                            </div>
                          )}

                        {/* Raw AI Response */}
                        {aiTestResult.ai_response && (
                          <div>
                            <div className="flex items-center gap-1.5 mb-1.5">
                              <FileText className="w-3 h-3 text-nofx-text-muted" />
                              <span className="text-xs font-medium text-nofx-text">
                                {t('aiOutput')} (Raw)
                              </span>
                            </div>
                            <pre
                              className="p-2 rounded-lg text-[10px] font-mono overflow-auto whitespace-pre-wrap bg-nofx-bg border border-nofx-gold/20 text-nofx-text"
                              style={{ maxHeight: '300px' }}
                            >
                              {aiTestResult.ai_response}
                            </pre>
                          </div>
                        )}

                        {/* Full Result JSON */}
                        <div className="pt-4 mt-4 border-t border-nofx-gold/10">
                          <div className="flex items-center gap-1.5 mb-1.5">
                            <Code className="w-3 h-3 text-nofx-text-muted" />
                            <span className="text-xs font-medium text-nofx-text">
                              Full Result JSON
                            </span>
                          </div>
                          <pre
                            className="p-2 rounded-lg text-[10px] font-mono overflow-auto bg-nofx-bg border border-nofx-gold/20 text-nofx-text-muted"
                            style={{ maxHeight: '300px' }}
                          >
                            {JSON.stringify(
                              (() => {
                                const { system_prompt, user_prompt, ...rest } = aiTestResult
                                return rest
                              })(),
                              null,
                              2
                            )}
                          </pre>
                        </div>
                      </>
                    )}
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center py-12 text-nofx-text-muted">
                    <Play className="w-10 h-10 mb-2 opacity-30" />
                    <p className="text-sm">
                      {language === 'zh'
                        ? '点击运行 AI 测试'
                        : 'Click to run AI test'}
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </DeepVoidBackground>
  )
}

export default ChaosStudioPage
