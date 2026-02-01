import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import {
  Calendar,
  TrendingUp,
  TrendingDown,
  Minus,
  MessageSquare,
  BarChart2,
  Brain,
} from 'lucide-react'
import type { AnalysisSession, AnalysisSessionDetail } from '../types'
import { notify } from '../lib/notify'
import { api } from '../lib/api'
import { DeepVoidBackground } from '../components/DeepVoidBackground'
import ReactMarkdown from 'react-markdown'

export function AnalysisDashboardPage() {
  const { token } = useAuth()
  const { language } = useLanguage()

  const [sessions, setSessions] = useState<AnalysisSession[]>([])
  const [selectedSessionId, setSelectedSessionId] = useState<number | null>(null)
  const [sessionDetail, setSessionDetail] = useState<AnalysisSessionDetail | null>(null)
  const [isLoadingList, setIsLoadingList] = useState(true)
  const [isLoadingDetail, setIsLoadingDetail] = useState(false)

  // Load session list
  const loadSessions = useCallback(async () => {
    if (!token) return
    try {
      const data = await api.getAnalysisSessions(20)
      setSessions(data)
      if (data.length > 0 && !selectedSessionId) {
        setSelectedSessionId(data[0].id)
      }
    } catch (err) {
      notify.error(language === 'zh' ? '加载会话列表失败' : 'Failed to load sessions')
    } finally {
      setIsLoadingList(false)
    }
  }, [token, language, selectedSessionId])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  // Load session detail
  useEffect(() => {
    const loadDetail = async () => {
      if (!selectedSessionId || !token) return
      setIsLoadingDetail(true)
      try {
        const detail = await api.getAnalysisSessionDetail(selectedSessionId)
        setSessionDetail(detail)
      } catch (err) {
        notify.error(language === 'zh' ? '加载详情失败' : 'Failed to load details')
      } finally {
        setIsLoadingDetail(false)
      }
    }
    loadDetail()
  }, [selectedSessionId, token, language])

  // Helper to format time
  const formatTime = (ts: number) => {
    return new Date(ts * 1000).toLocaleString(language === 'zh' ? 'zh-CN' : 'en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  // Helper for sentiment icon
  const SentimentIcon = ({ sentiment }: { sentiment: string }) => {
    const s = sentiment.toLowerCase()
    if (s.includes('bull')) return <TrendingUp className="w-5 h-5 text-green-500" />
    if (s.includes('bear')) return <TrendingDown className="w-5 h-5 text-red-500" />
    return <Minus className="w-5 h-5 text-gray-400" />
  }

  // Helper for sentiment color
  const getSentimentColor = (sentiment: string) => {
    const s = sentiment.toLowerCase()
    if (s.includes('bull')) return 'text-green-500 bg-green-500/10 border-green-500/20'
    if (s.includes('bear')) return 'text-red-500 bg-red-500/10 border-red-500/20'
    return 'text-gray-400 bg-gray-500/10 border-gray-500/20'
  }

  return (
    <DeepVoidBackground className="h-[calc(100vh-64px)] flex flex-col bg-nofx-bg relative overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-nofx-gold/20 bg-nofx-bg/60 backdrop-blur-md z-10">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-gradient-to-br from-purple-600 to-pink-600">
            <BarChart2 className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-lg font-bold text-nofx-text">
              {language === 'zh' ? '分析看板' : 'Analysis Dashboard'}
            </h1>
            <p className="text-xs text-nofx-text-muted">
              {language === 'zh' ? '多角度市场分析报告' : 'Multi-perspective market analysis reports'}
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar: Sessions List */}
        <div className="w-72 flex-shrink-0 border-r border-nofx-gold/20 overflow-y-auto bg-nofx-bg/30 backdrop-blur-sm z-10">
          <div className="p-3">
            <div className="text-xs font-medium text-nofx-text-muted mb-3 px-2 uppercase tracking-wider">
              {language === 'zh' ? '历史会话' : 'History'}
            </div>
            {isLoadingList ? (
              <div className="flex justify-center py-4">
                <div className="w-6 h-6 rounded-full border-2 border-purple-500/20 border-t-purple-500 animate-spin" />
              </div>
            ) : (
              <div className="space-y-2">
                {sessions.map(session => (
                  <div
                    key={session.id}
                    onClick={() => setSelectedSessionId(session.id)}
                    className={`p-3 rounded-lg cursor-pointer transition-all border ${
                      selectedSessionId === session.id
                        ? 'bg-purple-500/10 border-purple-500/50 shadow-[0_0_10px_rgba(168,85,247,0.1)]'
                        : 'bg-nofx-bg-lighter/40 border-transparent hover:bg-nofx-bg-lighter hover:border-nofx-gold/10'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs font-mono text-nofx-text-muted flex items-center gap-1">
                        <Calendar className="w-3 h-3" />
                        {formatTime(session.created_at)}
                      </span>
                      <span className="text-[10px] bg-white/5 px-1.5 py-0.5 rounded text-nofx-text-muted">
                        ID: {session.id}
                      </span>
                    </div>
                    <div className="text-sm text-nofx-text line-clamp-2">
                      {session.overall_summary || (language === 'zh' ? '无摘要' : 'No summary')}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Main Content: Detail View */}
        <div className="flex-1 overflow-y-auto bg-nofx-bg/50 p-6">
          {isLoadingDetail ? (
            <div className="flex items-center justify-center h-full">
              <div className="w-12 h-12 rounded-full border-4 border-purple-500/20 border-t-purple-500 animate-spin" />
            </div>
          ) : sessionDetail ? (
            <div className="max-w-5xl mx-auto space-y-8">
              
              {/* Market Context Summary */}
              <div className="bg-nofx-bg-lighter/80 backdrop-blur rounded-xl border border-nofx-gold/20 p-6 shadow-xl">
                <h2 className="text-xl font-bold text-nofx-text mb-4 flex items-center gap-2">
                  <Brain className="w-6 h-6 text-nofx-gold" />
                  {language === 'zh' ? '市场概况' : 'Market Context'}
                </h2>
                <div className="prose prose-invert prose-sm max-w-none">
                  {/* Assuming context_summary is a JSON string, we might want to parse it or just display key info if it's structured. 
                      If it's text, just display. For now, let's assume it might be JSON but we display the overall_summary more prominently. */}
                   <div className="bg-black/20 p-4 rounded-lg font-mono text-sm text-nofx-text-muted whitespace-pre-wrap">
                      {sessionDetail.session.context_summary}
                   </div>
                </div>
                
                {sessionDetail.session.overall_summary && (
                  <div className="mt-6 pt-6 border-t border-white/5">
                    <h3 className="text-sm font-medium text-nofx-text-muted mb-2 uppercase tracking-wider">
                      {language === 'zh' ? '会议总结' : 'Roundtable Summary'}
                    </h3>
                    <div className="text-lg text-nofx-text leading-relaxed">
                      {sessionDetail.session.overall_summary}
                    </div>
                  </div>
                )}
              </div>

              {/* Analyst Records Grid */}
              <div>
                <h3 className="text-lg font-medium text-nofx-text mb-4 flex items-center gap-2">
                  <MessageSquare className="w-5 h-5 text-purple-400" />
                  {language === 'zh' ? '分析师观点' : 'Analyst Opinions'}
                </h3>
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {sessionDetail.records.map(record => (
                    <div 
                      key={record.id} 
                      className="bg-nofx-bg-lighter border border-nofx-gold/10 rounded-xl overflow-hidden hover:border-nofx-gold/30 transition-colors"
                    >
                      {/* Card Header */}
                      <div className="px-5 py-4 border-b border-white/5 bg-white/5 flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-gray-700 to-gray-800 flex items-center justify-center border border-white/10">
                            <span className="text-xs font-bold text-white">{record.analyst_name.substring(0, 2).toUpperCase()}</span>
                          </div>
                          <div>
                            <div className="font-bold text-nofx-text">{record.analyst_name}</div>
                            <div className="text-xs text-nofx-text-muted font-mono opacity-70">
                                {new Date(record.created_at * 1000).toLocaleTimeString()}
                            </div>
                          </div>
                        </div>
                        <div className={`px-3 py-1 rounded-full text-xs font-medium border flex items-center gap-1.5 ${getSentimentColor(record.sentiment)}`}>
                          <SentimentIcon sentiment={record.sentiment} />
                          {record.sentiment}
                          <span className="ml-1 opacity-70 border-l border-current pl-1.5">
                            {record.score}/10
                          </span>
                        </div>
                      </div>

                      {/* Card Body */}
                      <div className="p-5">
                        <div className="prose prose-invert prose-sm max-w-none text-nofx-text-muted/90">
                          <ReactMarkdown>{record.reasoning}</ReactMarkdown>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-nofx-text-muted opacity-50">
              <BarChart2 className="w-16 h-16 mb-4" />
              <p>{language === 'zh' ? '选择一个会话查看详情' : 'Select a session to view details'}</p>
            </div>
          )}
        </div>
      </div>
    </DeepVoidBackground>
  )
}
