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
  Code
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

  // Fetch Records when trader selected
  useEffect(() => {
    if (!token || !selectedTraderId) return
    
    const fetchRecords = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/decisions?trader_id=${selectedTraderId}`, {
          headers: { Authorization: `Bearer ${token}` },
        })
        if (response.ok) {
          const data = await response.json()
          setRecords(data || [])
        }
      } catch (err) {
        console.error('Failed to fetch records:', err)
      }
    }
    
    fetchRecords()
  }, [token, selectedTraderId])

  // Initial load
  useEffect(() => {
    fetchAiModels()
    fetchTraders()
  }, [fetchAiModels, fetchTraders])

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
  }, [selectedRecord, traders, selectedTraderId, aiModels])

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
        <div className="w-64 flex-shrink-0 border-r border-nofx-gold/20 flex flex-col bg-nofx-bg/30 backdrop-blur-sm z-10">
          <div className="p-3 border-b border-nofx-gold/10">
            <label className="text-xs font-medium text-nofx-text-muted mb-2 block">{t('selectTrader')}</label>
            <select 
              value={selectedTraderId}
              onChange={(e) => setSelectedTraderId(e.target.value)}
              className="w-full px-2 py-1.5 rounded text-sm bg-nofx-bg border border-nofx-gold/20 text-nofx-text"
            >
              {traders.map(trader => (
                <option key={trader.trader_id} value={trader.trader_id}>{trader.trader_name}</option>
              ))}
            </select>
          </div>
          
          <div className="flex-1 overflow-y-auto p-2 space-y-1">
            {records.map(record => (
              <div
                key={record.id}
                onClick={() => setSelectedRecord(record)}
                className={`p-2 rounded cursor-pointer transition-colors text-xs ${
                  selectedRecord?.id === record.id 
                    ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' 
                    : 'hover:bg-white/5 text-nofx-text-muted'
                }`}
              >
                <div className="flex justify-between mb-1">
                  <span>{new Date(record.timestamp).toLocaleTimeString()}</span>
                  <span className={record.success ? 'text-green-500' : 'text-red-500'}>
                    {record.success ? 'Success' : 'Failed'}
                  </span>
                </div>
                <div className="text-[10px] opacity-70 truncate">
                  ID: {record.id} • Cycle: {record.cycle_number}
                </div>
              </div>
            ))}
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
                    className="px-2 py-1.5 rounded text-xs bg-nofx-bg border border-nofx-gold/20 text-nofx-text w-32"
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
                    {/* Quick Presets could go here */}
                  </div>
                  <textarea 
                    value={newSystemPrompt}
                    onChange={(e) => setNewSystemPrompt(e.target.value)}
                    className="flex-1 p-3 rounded-lg bg-black/40 border border-blue-500/30 text-xs text-nofx-text font-mono resize-none focus:outline-none focus:border-blue-500"
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
