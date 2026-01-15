import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import {
  Clock,
  Play,
  Activity,
  Bot,
  FileText,
  Sparkles,
  Layout,
  Code,
  Filter,
  ArrowDownUp,
  ChevronLeft,
  ChevronRight
} from 'lucide-react'
import { DeepVoidBackground } from '../components/DeepVoidBackground'
import { notify } from '../lib/notify'
import type { AIModel } from '../types'

const API_BASE = import.meta.env.VITE_API_BASE || ''

interface Trader {
  trader_id: string
  trader_name: string
  ai_model: string
  exchange_id: string
}

interface DecisionRecord {
  id: number
  trader_id: string
  cycle_number: number
  timestamp: string
  input_prompt: string
  system_prompt: string
  decision_json: string
  raw_response: string
  success: boolean
  decisions: any[]
}

export function TimeMachinePage() {
  const { token } = useAuth()
  const { language } = useLanguage()

  // State
  const [traders, setTraders] = useState<Trader[]>([])
  const [selectedTraderId, setSelectedTraderId] = useState<string>('')
  
  const [records, setRecords] = useState<DecisionRecord[]>([])
  const [selectedRecord, setSelectedRecord] = useState<DecisionRecord | null>(null)
  
  const [aiModels, setAiModels] = useState<AIModel[]>([])
  const [selectedModelId, setSelectedModelId] = useState<string>('')
  
  const [strategies, setStrategies] = useState<any[]>([])
  const [selectedStrategyId, setSelectedStrategyId] = useState<string>('')
  const [selectedVariant, setSelectedVariant] = useState<string>('balanced')

  // Filter & Sort State
  const [filterType, setFilterType] = useState<'all' | 'failed' | 'has_trades'>('all')
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc')
  
  // Pagination State
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)

  const [newSystemPrompt, setNewSystemPrompt] = useState<string>('')
  const [isRunning, setIsRunning] = useState(false)
  
  const [timeMachineResult, setTimeMachineResult] = useState<{
    ai_response: string
    decisions: any[]
    reasoning?: string
  } | null>(null)

  // Fetch AI Models
  const fetchAiModels = useCallback(async () => {
    if (!token) return
    try {
      const response = await fetch(`${API_BASE}/api/models`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        const allModels = Array.isArray(data) ? data : (data.models || [])
        const enabledModels = allModels.filter((m: AIModel) => m.enabled)
        setAiModels(enabledModels)
      }
    } catch (err) {
      console.error('Failed to fetch AI models:', err)
    }
  }, [token])

  // Fetch Traders
  const fetchTraders = useCallback(async () => {
    if (!token) return
    try {
      const response = await fetch(`${API_BASE}/api/my-traders`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        setTraders(data || [])
        if (data.length > 0) {
          setSelectedTraderId(data[0].trader_id)
        }
      }
    } catch (err) {
      console.error('Failed to fetch traders:', err)
    }
  }, [token])

  // Fetch Records when trader selected (with pagination)
  useEffect(() => {
    if (!token || !selectedTraderId) return
    
    const fetchRecords = async () => {
      try {
        const query = new URLSearchParams({
            trader_id: selectedTraderId,
            page: page.toString(),
            page_size: pageSize.toString(),
            filter: filterType,
            sort: sortOrder
        })
        const response = await fetch(`${API_BASE}/api/decisions?${query}`, {
          headers: { Authorization: `Bearer ${token}` },
        })
        if (response.ok) {
          const data = await response.json()
          setRecords(data.items || [])
          setTotal(data.total || 0)
        }
      } catch (err) {
        console.error('Failed to fetch records:', err)
      }
    }
    
    fetchRecords()
  }, [token, selectedTraderId, page, pageSize, filterType, sortOrder])

  // Reset page when filter/trader changes
  useEffect(() => {
      setPage(1)
  }, [filterType, selectedTraderId, sortOrder])

  // Fetch Strategies
  const fetchStrategies = useCallback(async () => {
    if (!token) return
    try {
      const response = await fetch(`${API_BASE}/api/strategies`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        setStrategies(data.strategies || [])
      }
    } catch (err) {
      console.error('Failed to fetch strategies:', err)
    }
  }, [token])

  // Initial load
  useEffect(() => {
    fetchAiModels()
    fetchTraders()
    fetchStrategies()
  }, [fetchAiModels, fetchTraders, fetchStrategies])

  // When record selected, set prompt and model
  useEffect(() => {
    if (selectedRecord) {
      setNewSystemPrompt(selectedRecord.system_prompt)
      setTimeMachineResult(null)
       
      // Try to match trader's model
      const trader = traders.find(t => t.trader_id === selectedTraderId)
      if (trader) {
        // Find matching model ID (trader.ai_model might be simple id or full id)
        const model = aiModels.find(m => m.id === trader.ai_model || trader.ai_model.includes(m.id))
        if (model) setSelectedModelId(model.id)
        else if (aiModels.length > 0) setSelectedModelId(aiModels[0].id)
      }
    }
  }, [selectedRecord?.id]) // Only trigger when the ID changes

  // Update model if models load later
  useEffect(() => {
    if (selectedRecord && !selectedModelId && aiModels.length > 0) {
       const trader = traders.find(t => t.trader_id === selectedTraderId)
       if (trader) {
         const model = aiModels.find(m => m.id === trader.ai_model || trader.ai_model.includes(m.id))
         if (model) setSelectedModelId(model.id)
         else setSelectedModelId(aiModels[0].id)
       }
    }
  }, [aiModels, selectedRecord, selectedTraderId, selectedModelId, traders])

  // Computed displayed records (removed, using server-side pagination)

  const handleGeneratePrompt = async () => {
    if (!token || !selectedStrategyId) return

    const strategy = strategies.find(s => s.id === selectedStrategyId)
    if (!strategy) return

    try {
      const response = await fetch(`${API_BASE}/api/strategies/preview-prompt`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          config: strategy.config,
          prompt_variant: selectedVariant,
          account_equity: 1000 // Default or mocked equity
        }),
      })

      if (response.ok) {
        const data = await response.json()
        setNewSystemPrompt(data.system_prompt)
        notify.success(language === 'zh' ? 'Prompt 已生成' : 'Prompt generated')
      } else {
        throw new Error('Failed to generate prompt')
      }
    } catch (err) {
      console.error(err)
      notify.error(language === 'zh' ? '生成 Prompt 失败' : 'Failed to generate prompt')
    }
  }

  const handleRunTimeMachine = async () => {
    if (!token || !selectedRecord || !selectedModelId) return
    
    setIsRunning(true)
    try {
      const response = await fetch(`${API_BASE}/api/strategies/time-machine`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          decision_id: selectedRecord.id,
          system_prompt: newSystemPrompt,
          ai_model_id: selectedModelId,
        }),
      })
      
      if (!response.ok) throw new Error('Time Machine failed')
      
      const data = await response.json()
      setTimeMachineResult(data)
      notify.success(language === 'zh' ? '时光机运行成功' : 'Time Machine run successful')
    } catch (err) {
      notify.error(language === 'zh' ? '运行时光机失败' : 'Failed to run Time Machine')
      console.error(err)
    } finally {
      setIsRunning(false)
    }
  }

  const t = (key: string) => {
    const dict: any = {
      title: { zh: '时光机', en: 'Time Machine' },
      subtitle: { zh: '重放历史交易决策', en: 'Replay historical trading decisions' },
      traders: { zh: '交易员', en: 'Traders' },
      records: { zh: '交易记录', en: 'Transactions' },
      selectTrader: { zh: '选择交易员', en: 'Select Trader' },
      selectRecord: { zh: '选择交易记录', en: 'Select Transaction' },
      systemPrompt: { zh: 'System Prompt (可编辑)', en: 'System Prompt (Editable)' },
      userPrompt: { zh: 'User Prompt (原始数据)', en: 'User Prompt (Original Data)' },
      run: { zh: '启动时光机', en: 'Run Time Machine' },
      running: { zh: '穿越中...', en: 'Traveling...' },
      originalDecision: { zh: '原始决策', en: 'Original Decision' },
      newDecision: { zh: '新决策', en: 'New Decision' },
      reasoning: { zh: '推理过程', en: 'Reasoning' },
      model: { zh: 'AI 模型', en: 'AI Model' },
      presets: { zh: '预设', en: 'Presets' },
      selectStrategy: { zh: '选择策略模板', en: 'Select Strategy' },
      generate: { zh: '生成 Prompt', en: 'Generate' },
      filterAll: { zh: '全部', en: 'All' },
      filterFailed: { zh: '失败', en: 'Failed' },
      filterHasTrades: { zh: '有交易', en: 'Has Trades' },
    }
    return dict[key]?.[language] || key
  }

  return (
    <DeepVoidBackground className="h-[calc(100vh-64px)] flex flex-col bg-nofx-bg relative overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-nofx-gold/20 bg-nofx-bg/60 backdrop-blur-md z-10">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-500">
            <Clock className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-lg font-bold text-nofx-text">{t('title')}</h1>
            <p className="text-xs text-nofx-text-muted">{t('subtitle')}</p>
          </div>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Left Column: Traders & Records */}
        <div className="w-96 flex-shrink-0 border-r border-nofx-gold/20 flex flex-col bg-nofx-bg/30 backdrop-blur-sm z-10">
          <div className="p-3 border-b border-nofx-gold/10">
            <label className="text-xs font-medium text-nofx-text-muted mb-2 block">{t('selectTrader')}</label>
            <select 
              value={selectedTraderId}
              onChange={(e) => setSelectedTraderId(e.target.value)}
              disabled={isRunning}
              className="w-full px-2 py-1.5 rounded text-sm bg-nofx-bg border border-nofx-gold/20 text-nofx-text disabled:opacity-50"
            >
              {traders.map(trader => (
                <option key={trader.trader_id} value={trader.trader_id}>{trader.trader_name}</option>
              ))}
            </select>
          </div>
          
          {/* Filter & Sort Toolbar */}
          <div className="px-3 py-2 border-b border-nofx-gold/10 flex items-center justify-between gap-2">
            <div className="flex items-center gap-1 flex-1">
              <Filter className="w-3 h-3 text-nofx-text-muted" />
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as any)}
                disabled={isRunning}
                className="w-full px-1 py-1 rounded text-[10px] bg-nofx-bg border border-nofx-gold/20 text-nofx-text disabled:opacity-50"
              >
                <option value="all">{t('filterAll')}</option>
                <option value="failed">{t('filterFailed')}</option>
                <option value="has_trades">{t('filterHasTrades')}</option>
              </select>
            </div>
            <button
              onClick={() => !isRunning && setSortOrder(sortOrder === 'desc' ? 'asc' : 'desc')}
              disabled={isRunning}
              className={`p-1 rounded hover:bg-white/10 transition-colors ${isRunning ? 'opacity-50 cursor-not-allowed' : ''}`}
              title={sortOrder === 'desc' ? 'Newest First' : 'Oldest First'}
            >
              <ArrowDownUp className={`w-3 h-3 ${sortOrder === 'desc' ? 'text-nofx-gold' : 'text-nofx-text-muted'}`} />
            </button>
          </div>
          
          <div className="flex-1 overflow-y-auto p-2 space-y-2">
            {records.map(record => (
              <div
                key={record.id}
                onClick={() => !isRunning && setSelectedRecord(record)}
                className={`p-3 rounded cursor-pointer transition-colors text-xs border ${
                  selectedRecord?.id === record.id 
                    ? 'bg-blue-500/20 text-blue-400 border-blue-500/30' 
                    : 'bg-white/5 border-white/5 hover:bg-white/10 text-nofx-text-muted'
                } ${isRunning ? 'opacity-50 cursor-not-allowed pointer-events-none' : ''}`}
              >
                <div className="flex justify-between items-center mb-1">
                   <span className="font-mono opacity-70">{new Date(record.timestamp).toLocaleString()}</span>
                   <span className={`px-1.5 py-0.5 rounded text-[10px] ${record.success ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
                      {record.success ? 'Success' : 'Fail'}
                   </span>
                </div>
                
                <div className="flex justify-between items-center mb-1">
                    <span className="font-bold text-nofx-text">Cycle #{record.cycle_number}</span>
                    <span className="text-[10px] opacity-50">ID: {record.id}</span>
                </div>

                {/* Decisions Summary */}
                {record.decisions && record.decisions.length > 0 ? (
                    <div className="flex flex-wrap gap-1 mt-2">
                        {record.decisions.map((d, i) => (
                            <span key={i} className={`px-1.5 py-0.5 rounded text-[10px] border ${
                                d.action.includes('open') ? 'bg-green-500/10 border-green-500/20 text-green-300' :
                                d.action.includes('close') ? 'bg-orange-500/10 border-orange-500/20 text-orange-300' :
                                'bg-white/5 border-white/10 text-nofx-text-muted'
                            }`}>
                                {d.action} {d.symbol}
                            </span>
                        ))}
                    </div>
                ) : (
                    <div className="mt-1 text-[10px] opacity-40 italic">No actions recorded</div>
                )}
              </div>
            ))}
          </div>

          {/* Pagination */}
          <div className="p-2 border-t border-nofx-gold/10 flex items-center justify-between text-xs text-nofx-text-muted bg-nofx-bg">
              <button 
                  onClick={() => !isRunning && setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1 || isRunning}
                  className="p-1 hover:text-white disabled:opacity-30 transition-colors"
              >
                  <ChevronLeft className="w-4 h-4" />
              </button>
              <span>Page {page} of {Math.ceil(total / pageSize) || 1} ({total} items)</span>
              <button 
                  onClick={() => !isRunning && setPage(p => p + 1)}
                  disabled={page >= Math.ceil(total / pageSize) || isRunning}
                  className="p-1 hover:text-white disabled:opacity-30 transition-colors"
              >
                  <ChevronRight className="w-4 h-4" />
              </button>
          </div>
        </div>

        {/* Middle Column: Prompt Editor */}
        <div className="flex-1 min-w-0 flex flex-col border-r border-nofx-gold/20 bg-nofx-bg/10">
          {selectedRecord ? (
            <div className="flex-1 flex flex-col h-full">
              {/* Toolbar */}
              <div className="p-3 border-b border-nofx-gold/10 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Bot className="w-4 h-4 text-nofx-gold" />
                  <span className="text-sm font-medium text-nofx-text">
                    ID: {selectedRecord.id} ({new Date(selectedRecord.timestamp).toLocaleString()})
                  </span>
                </div>
                
                <div className="flex items-center gap-2">
                   <select 
                    value={selectedModelId}
                    onChange={(e) => setSelectedModelId(e.target.value)}
                    disabled={isRunning}
                    className="px-2 py-1.5 rounded text-xs bg-nofx-bg border border-nofx-gold/20 text-nofx-text w-32 disabled:opacity-50"
                  >
                    {aiModels.map(m => (
                      <option key={m.id} value={m.id}>{m.name}</option>
                    ))}
                  </select>
                  
                  <button
                    onClick={handleRunTimeMachine}
                    disabled={isRunning}
                    className="flex items-center gap-2 px-4 py-1.5 rounded bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium transition-colors disabled:opacity-50"
                  >
                    {isRunning ? (
                      <Activity className="w-3 h-3 animate-spin" />
                    ) : (
                      <Play className="w-3 h-3" />
                    )}
                    {isRunning ? t('running') : t('run')}
                  </button>
                </div>
              </div>

              {/* Editor Area */}
              <div className="flex-1 overflow-hidden flex flex-col p-4 gap-4">
                {/* User Prompt (Read Only) */}
                <div className="h-1/3 flex flex-col">
                  <div className="flex items-center gap-2 mb-2">
                    <Layout className="w-3 h-3 text-nofx-text-muted" />
                    <span className="text-xs font-medium text-nofx-text-muted">{t('userPrompt')}</span>
                  </div>
                  <pre className="flex-1 p-3 rounded-lg bg-black/40 border border-nofx-gold/10 text-[10px] text-nofx-text-muted font-mono overflow-auto resize-none">
                    {selectedRecord.input_prompt}
                  </pre>
                </div>

                {/* System Prompt (Editable) */}
                <div className="flex-1 flex flex-col">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Code className="w-3 h-3 text-blue-400" />
                      <span className="text-xs font-medium text-blue-400">{t('systemPrompt')}</span>
                    </div>
                    
                    {/* Strategy Selector */}
                    <div className="flex items-center gap-2">
                      <select
                        value={selectedStrategyId}
                        onChange={(e) => setSelectedStrategyId(e.target.value)}
                        disabled={isRunning}
                        className="px-2 py-1 rounded text-[10px] bg-nofx-bg border border-nofx-gold/20 text-nofx-text w-32 disabled:opacity-50"
                      >
                        <option value="">{t('selectStrategy')}</option>
                        {strategies.map(s => (
                          <option key={s.id} value={s.id}>{s.name}</option>
                        ))}
                      </select>
                      
                      <select
                        value={selectedVariant}
                        onChange={(e) => setSelectedVariant(e.target.value)}
                        disabled={isRunning}
                        className="px-2 py-1 rounded text-[10px] bg-nofx-bg border border-nofx-gold/20 text-nofx-text w-24 disabled:opacity-50"
                      >
                        <option value="balanced">Balanced</option>
                        <option value="aggressive">Aggressive</option>
                        <option value="conservative">Conservative</option>
                      </select>
                      
                      <button
                        onClick={handleGeneratePrompt}
                        disabled={!selectedStrategyId || isRunning}
                        className="px-2 py-1 rounded bg-white/10 hover:bg-white/20 text-[10px] text-nofx-text disabled:opacity-50"
                      >
                        {t('generate')}
                      </button>
                    </div>
                  </div>
                  <textarea 
                    value={newSystemPrompt}
                    onChange={(e) => setNewSystemPrompt(e.target.value)}
                    disabled={isRunning}
                    className="flex-1 p-3 rounded-lg bg-black/40 border border-blue-500/30 text-xs text-nofx-text font-mono resize-none focus:outline-none focus:border-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                    spellCheck={false}
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center text-nofx-text-muted">
              <div className="text-center">
                <Clock className="w-12 h-12 mx-auto mb-3 opacity-20" />
                <p>{t('selectRecord')}</p>
              </div>
            </div>
          )}
        </div>

        {/* Right Column: Comparison */}
        <div className="w-[450px] flex-shrink-0 flex flex-col bg-nofx-bg/30 backdrop-blur-sm z-10 overflow-y-auto">
          {selectedRecord && (
            <div className="p-4 space-y-6">
              {/* Original Result */}
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-nofx-text-muted">
                  <FileText className="w-4 h-4" />
                  <span className="text-sm font-medium">{t('originalDecision')}</span>
                </div>
                <div className="p-3 rounded-lg bg-white/5 border border-white/10 space-y-2">
                  <div className="flex flex-wrap gap-2">
                    {selectedRecord.decisions && selectedRecord.decisions.map((d, i) => (
                      <span key={i} className="px-2 py-1 rounded bg-white/10 text-[10px] text-nofx-text">
                        {d.action} {d.symbol}
                      </span>
                    ))}
                  </div>
                  {/* Parse original reasoning if possible, or show snippet */}
                  {/* Since DB stores parsing separate from raw response, we might not have pure reasoning string easily unless we parse JSON or use raw */}
                  <div className="text-[10px] text-nofx-text-muted/70 max-h-32 overflow-auto whitespace-pre-wrap">
                    {selectedRecord.raw_response}
                  </div>
                </div>
              </div>

              {/* New Result */}
              {timeMachineResult && (
                <div className="space-y-2 animate-in fade-in slide-in-from-bottom-4 duration-500">
                  <div className="flex items-center gap-2 text-blue-400">
                    <Sparkles className="w-4 h-4" />
                    <span className="text-sm font-bold">{t('newDecision')}</span>
                  </div>
                  <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/30 space-y-2">
                     <div className="flex flex-wrap gap-2">
                      {timeMachineResult.decisions && timeMachineResult.decisions.map((d, i) => (
                        <span key={i} className="px-2 py-1 rounded bg-blue-500/20 text-[10px] text-blue-300 border border-blue-500/30">
                          {d.action} {d.symbol}
                        </span>
                      ))}
                    </div>
                    <div className="text-[10px] text-nofx-text max-h-96 overflow-auto whitespace-pre-wrap font-mono">
                      {timeMachineResult.ai_response}
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </DeepVoidBackground>
  )
}

export default TimeMachinePage
