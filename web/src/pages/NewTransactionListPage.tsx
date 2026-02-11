import { useState } from 'react'
import useSWR from 'swr'
import { api } from '../lib/api'
import { DeepVoidBackground } from '../components/DeepVoidBackground'
import { useLanguage } from '../contexts/LanguageContext'
import type { TraderInfo, Exchange } from '../types'
import { PositionHistoryTab } from '../components/PositionHistoryTab'
import { TransactionHistoryTab } from '../components/TransactionHistoryTab'
import { BarChart3 } from 'lucide-react'

export function NewTransactionListPage() {
  const { language } = useLanguage()
  const [activeTab, setActiveTab] = useState<'positions' | 'transactions'>('positions')
  const [selectedTraderId, setSelectedTraderId] = useState<string>('')
  
  // Fetch lists for filters
  const { data: traders } = useSWR<TraderInfo[]>('traders', api.getTraders)
  const { data: exchanges } = useSWR<Exchange[]>('exchanges', api.getExchangeConfigs)

  // Get selected trader name for display
  const selectedTrader = traders?.find(t => t.trader_id === selectedTraderId)
  const selectedTraderName = selectedTrader ? selectedTrader.trader_name : ''

  return (
    <DeepVoidBackground className="h-[calc(100vh-64px)] flex flex-col bg-nofx-bg relative overflow-hidden">
      {/* Header - Match ChaosStudio style */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-nofx-gold/20 bg-nofx-bg/60 backdrop-blur-md z-10">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-gradient-to-br from-purple-600 to-indigo-600">
              <BarChart3 className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-nofx-text">
                {language === 'zh' ? '交易列表' : 'Transaction List'}
              </h1>
              {selectedTraderName ? (
                <p className="text-xs text-nofx-text-muted">
                  {language === 'zh' 
                    ? `查看 ${selectedTraderName} 的历史仓位和成交记录` 
                    : `View historical positions and transaction records for ${selectedTraderName}`}
                </p>
              ) : (
                <p className="text-xs text-nofx-text-muted">
                  {language === 'zh' 
                    ? '请选择交易员查看历史仓位和成交记录' 
                    : 'Please select a trader to view historical positions and transaction records'}
                </p>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Main Content - Match ChaosStudio three-column layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar - Trader Selection */}
        <div className="w-64 flex-shrink-0 border-r border-nofx-gold/20 overflow-y-auto bg-nofx-bg/30 backdrop-blur-sm z-10">
          <div className="p-3">
            <div className="flex items-center justify-between mb-3 px-2">
              <h2 className="text-sm font-semibold text-nofx-text-muted uppercase tracking-wide">
                {language === 'zh' ? '交易员' : 'Traders'}
              </h2>
            </div>
            <div className="space-y-1">
              {traders?.map(trader => (
                <button
                  key={trader.trader_id}
                  onClick={() => setSelectedTraderId(trader.trader_id)}
                  className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-all group ${
                    selectedTraderId === trader.trader_id
                      ? 'ring-1 ring-purple-500/50 bg-purple-500/10 shadow-[0_0_15px_rgba(168,85,247,0.1)]'
                      : 'hover:bg-nofx-bg-lighter/60 hover:ring-1 hover:ring-purple-500/20 bg-transparent'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${selectedTraderId === trader.trader_id ? 'bg-purple-500' : 'bg-nofx-text-muted/30'}`}></div>
                    <span className={selectedTraderId === trader.trader_id ? 'text-nofx-text' : 'text-nofx-text-muted'}>
                      {trader.trader_name}
                    </span>
                  </div>
                </button>
              ))}
            </div>
            
            {traders && traders.length === 0 && (
              <div className="text-center py-6 text-nofx-text-muted text-sm">
                {language === 'zh' ? '暂无交易员' : 'No traders available'}
              </div>
            )}
          </div>
        </div>

        {/* Right Content Area */}
        <div className="flex-1 flex flex-col overflow-hidden bg-nofx-bg/20">
          {/* Tab Navigation - Match ChaosStudio style */}
          <div className="flex-shrink-0 px-4 py-3 border-b border-nofx-gold/10 bg-nofx-bg/40 backdrop-blur-sm">
            <div className="flex space-x-1 bg-black/30 p-1 rounded-lg border border-white/10 w-fit">
              <button
                onClick={() => setActiveTab('positions')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 ${
                  activeTab === 'positions'
                    ? 'bg-purple-600 text-white shadow-[0_0_15px_rgba(168,85,247,0.3)]'
                    : 'text-nofx-text-muted hover:text-nofx-text hover:bg-white/5'
                }`}
              >
                {language === 'zh' ? '历史仓位' : 'Position History'}
              </button>
              <button
                onClick={() => setActiveTab('transactions')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 ${
                  activeTab === 'transactions'
                    ? 'bg-purple-600 text-white shadow-[0_0_15px_rgba(168,85,247,0.3)]'
                    : 'text-nofx-text-muted hover:text-nofx-text hover:bg-white/5'
                }`}
              >
                {language === 'zh' ? '成交记录' : 'Transaction History'}
              </button>
            </div>
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-auto">
            <div className="p-4 h-full">
              {selectedTraderId ? (
                activeTab === 'positions' ? (
                  <PositionHistoryTab 
                    traderId={selectedTraderId} 
                    traderName={selectedTraderName}
                  />
                ) : (
                  <TransactionHistoryTab 
                    traderId={selectedTraderId} 
                    traderName={selectedTraderName}
                    exchanges={exchanges || []}
                  />
                )
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-nofx-text-muted">
                  <BarChart3 className="w-12 h-12 mb-4 opacity-30" />
                  <div className="text-lg font-semibold mb-2">
                    {language === 'zh' ? '请选择交易员' : 'Please Select a Trader'}
                  </div>
                  <div className="text-sm max-w-md text-center">
                    {language === 'zh' 
                      ? '从左侧边栏选择一个交易员开始查看数据' 
                      : 'Select a trader from the left sidebar to start viewing data'}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </DeepVoidBackground>
  )
}