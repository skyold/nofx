import { useState } from 'react'
import useSWR, { mutate } from 'swr'
import { api } from '../lib/api'
import { DeepVoidBackground } from '../components/DeepVoidBackground'
import { notify } from '../lib/notify'
import { useLanguage } from '../contexts/LanguageContext'
import { Loader2, ArrowUpDown } from 'lucide-react'
import type { TraderInfo, Exchange } from '../types'

export function TransactionListPage() {
  const { language } = useLanguage()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [exchangeId, setExchangeId] = useState<string>('all')
  const [traderId, setTraderId] = useState<string>('all')
  const [sort, setSort] = useState<'desc' | 'asc'>('desc')
  const [assigningId, setAssigningId] = useState<number | null>(null)

  // Fetch lists for filters
  const { data: traders } = useSWR<TraderInfo[]>('traders', api.getTraders)
  const { data: exchanges } = useSWR<Exchange[]>('exchanges', api.getExchangeConfigs)

  // Fetch transactions
  const { data: transactionData, error, isLoading } = useSWR(
    [`transactions`, page, pageSize, exchangeId, traderId, sort],
    () => api.getTransactions(page, pageSize, exchangeId, traderId, sort)
  )

  const handleAssignTrader = async (transactionId: number, newTraderId: string) => {
    if (!newTraderId) return
    setAssigningId(transactionId)
    try {
      await api.assignTransaction(transactionId, newTraderId)
      notify.success(language === 'zh' ? '分配成功' : 'Assigned successfully')
      mutate([`transactions`, page, pageSize, exchangeId, traderId, sort])
    } catch (err) {
      notify.error(language === 'zh' ? '分配失败' : 'Failed to assign')
    } finally {
      setAssigningId(null)
    }
  }

  const toggleSort = () => {
    setSort(prev => prev === 'desc' ? 'asc' : 'desc')
  }

  const totalPages = transactionData ? Math.ceil(transactionData.total / pageSize) : 0

  return (
    <DeepVoidBackground className="min-h-screen pb-12 pt-6" disableAnimation>
      <div className="w-full px-4 md:px-8 relative z-10">
        <div className="mb-6 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
          <h1 className="text-2xl font-bold text-nofx-text-main">
            {language === 'zh' ? '交易列表' : 'Transaction List'}
          </h1>
          
          <div className="flex flex-wrap gap-4">
            {/* Exchange Filter */}
            <select
              value={exchangeId}
              onChange={(e) => { setExchangeId(e.target.value); setPage(1); }}
              className="bg-black/40 border border-white/10 rounded px-3 py-2 text-sm text-nofx-text-main focus:outline-none focus:border-nofx-gold/50"
            >
              <option value="all">{language === 'zh' ? '所有交易所' : 'All Exchanges'}</option>
              {exchanges?.map(ex => (
                <option key={ex.id} value={ex.id}>{ex.account_name || ex.name}</option>
              ))}
            </select>

            {/* Trader Filter */}
            <select
              value={traderId}
              onChange={(e) => { setTraderId(e.target.value); setPage(1); }}
              className="bg-black/40 border border-white/10 rounded px-3 py-2 text-sm text-nofx-text-main focus:outline-none focus:border-nofx-gold/50"
            >
              <option value="all">{language === 'zh' ? '所有交易员' : 'All Traders'}</option>
              {traders?.map(tr => (
                <option key={tr.trader_id} value={tr.trader_id}>{tr.trader_name}</option>
              ))}
            </select>
          </div>
        </div>

        <div className="nofx-glass p-6 overflow-hidden">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-nofx-gold" />
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">
              {language === 'zh' ? '加载失败' : 'Failed to load'}
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="text-xs text-nofx-text-muted uppercase bg-white/5 border-b border-white/10">
                    <tr>
                      <th className="px-4 py-3">ID</th>
                      <th 
                        className="px-4 py-3 cursor-pointer hover:text-white flex items-center gap-1"
                        onClick={toggleSort}
                      >
                        {language === 'zh' ? '时间' : 'Time'}
                        <ArrowUpDown className="w-3 h-3" />
                      </th>
                      <th className="px-4 py-3">{language === 'zh' ? '订单号' : 'Order ID'}</th>
                      <th className="px-4 py-3">{language === 'zh' ? '成交号' : 'Trade ID'}</th>
                      <th className="px-4 py-3">{language === 'zh' ? '交易所' : 'Exchange'}</th>
                      <th className="px-4 py-3">{language === 'zh' ? '交易对' : 'Symbol'}</th>
                      <th className="px-4 py-3">{language === 'zh' ? '方向' : 'Side'}</th>
                      <th className="px-4 py-3">{language === 'zh' ? '类型' : 'Type'}</th>
                      <th className="px-4 py-3 text-right">{language === 'zh' ? '价格' : 'Price'}</th>
                      <th className="px-4 py-3 text-right">{language === 'zh' ? '数量' : 'Quantity'}</th>
                      <th className="px-4 py-3 text-right">{language === 'zh' ? '成交额' : 'Value'}</th>
                      <th className="px-4 py-3 text-right">{language === 'zh' ? '手续费' : 'Fee'}</th>
                      <th className="px-4 py-3 text-right">{language === 'zh' ? '已实现盈亏' : 'Realized PnL'}</th>
                      <th className="px-4 py-3">{language === 'zh' ? '交易员' : 'Trader'}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {transactionData?.items.map((tx) => (
                      <tr key={tx.id} className="border-b border-white/5 hover:bg-white/5 transition-colors">
                        <td className="px-4 py-3 font-mono text-xs opacity-60">{tx.id}</td>
                        <td className="px-4 py-3 text-nofx-text-muted whitespace-nowrap">
                          {new Date(tx.created_at).toLocaleString()}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-nofx-text-muted" title={tx.exchange_order_id}>
                          {tx.exchange_order_id.length > 12 ? tx.exchange_order_id.slice(0, 8) + '...' : tx.exchange_order_id}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-nofx-text-muted" title={tx.exchange_trade_id}>
                          {tx.exchange_trade_id.length > 12 ? tx.exchange_trade_id.slice(0, 8) + '...' : tx.exchange_trade_id}
                        </td>
                        <td className="px-4 py-3">
                          <span className="px-2 py-0.5 rounded bg-white/5 text-xs border border-white/10">
                             {exchanges?.find(e => e.id === tx.exchange_id)?.account_name || tx.exchange_type}
                          </span>
                        </td>
                        <td className="px-4 py-3 font-mono font-bold text-nofx-text-main">{tx.symbol}</td>
                        <td className="px-4 py-3">
                           <span className={`px-1.5 py-0.5 rounded text-xs font-bold uppercase ${
                             tx.side.toUpperCase() === 'BUY' ? 'bg-nofx-green/10 text-nofx-green' : 'bg-nofx-red/10 text-nofx-red'
                           }`}>
                             {tx.side}
                           </span>
                        </td>
                        <td className="px-4 py-3">
                          <span className="px-1.5 py-0.5 rounded text-xs bg-white/5 text-nofx-text-muted">
                            {tx.is_maker ? 'MAKER' : 'TAKER'}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right font-mono">{tx.price.toFixed(4)}</td>
                        <td className="px-4 py-3 text-right font-mono">{tx.quantity.toFixed(4)}</td>
                        <td className="px-4 py-3 text-right font-mono">{tx.quote_quantity.toFixed(2)}</td>
                        <td className="px-4 py-3 text-right font-mono text-nofx-text-muted">{tx.commission.toFixed(4)}</td>
                        <td className="px-4 py-3 text-right font-mono">
                          {tx.realized_pnl !== 0 && (
                             <span className={tx.realized_pnl > 0 ? 'text-nofx-green' : 'text-nofx-red'}>
                               {tx.realized_pnl > 0 ? '+' : ''}{tx.realized_pnl.toFixed(2)}
                             </span>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                             <select
                               value={tx.trader_id || ""}
                               onChange={(e) => handleAssignTrader(tx.id, e.target.value)}
                               disabled={assigningId === tx.id}
                               className="bg-black/20 border border-white/10 rounded px-2 py-1 text-xs focus:outline-none focus:border-nofx-gold/50 max-w-[120px]"
                             >
                               <option value="">{language === 'zh' ? '未分配' : 'Unassigned'}</option>
                               {traders?.map(tr => (
                                 <option key={tr.trader_id} value={tr.trader_id}>{tr.trader_name}</option>
                               ))}
                             </select>
                             {assigningId === tx.id && <Loader2 className="w-3 h-3 animate-spin" />}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              <div className="flex flex-wrap justify-between items-center mt-6 text-sm text-nofx-text-muted gap-4">
                <div className="flex items-center gap-4">
                  <span>
                    {language === 'zh'
                      ? `显示 ${(page - 1) * pageSize + 1} 到 ${Math.min(page * pageSize, transactionData?.total || 0)} 条，共 ${transactionData?.total || 0} 条`
                      : `Showing ${(page - 1) * pageSize + 1} to ${Math.min(page * pageSize, transactionData?.total || 0)} of ${transactionData?.total || 0}`}
                  </span>
                  
                  {/* Page Size Selector */}
                  <div className="flex items-center gap-2">
                    <span>{language === 'zh' ? '每页' : 'Per page'}:</span>
                    <select
                      value={pageSize}
                      onChange={(e) => {
                        setPageSize(Number(e.target.value));
                        setPage(1);
                      }}
                      className="bg-black/40 border border-white/10 rounded px-2 py-1 text-xs focus:outline-none focus:border-nofx-gold/50"
                    >
                      <option value={20}>20</option>
                      <option value={50}>50</option>
                      <option value={100}>100</option>
                    </select>
                  </div>
                </div>

                <div className="flex gap-2">
                  <button
                    onClick={() => setPage(p => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="px-3 py-1 rounded bg-white/5 hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {language === 'zh' ? '上一页' : 'Previous'}
                  </button>
                  <span className="px-3 py-1">{page} / {totalPages}</span>
                  <button
                    onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="px-3 py-1 rounded bg-white/5 hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {language === 'zh' ? '下一页' : 'Next'}
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </DeepVoidBackground>
  )
}
