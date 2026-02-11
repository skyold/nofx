import { useState } from 'react'
import useSWR from 'swr'
import { api } from '../lib/api'
import { useLanguage } from '../contexts/LanguageContext'
import { Loader2, ArrowUpDown } from 'lucide-react'
import type { Exchange, Transaction } from '../types'

interface TransactionHistoryTabProps {
  traderId: string
  traderName: string
  exchanges: Exchange[]
}

export function TransactionHistoryTab({ traderId, traderName, exchanges }: TransactionHistoryTabProps) {
  const { language } = useLanguage()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [exchangeId, setExchangeId] = useState<string>('all')
  // traderId is now passed as prop, no longer managed locally
  const [sort, setSort] = useState<'desc' | 'asc'>('desc')

  // Fetch transactions for the selected trader
  const { data: transactionData, error, isLoading, isValidating } = useSWR(
    traderId ? [`transactions-tab`, page, pageSize, exchangeId, traderId, sort] : null,
    () => traderId ? api.getTransactions(page, pageSize, exchangeId, traderId, sort) : Promise.resolve(null),
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  )

  const toggleSort = () => {
    setSort(prev => prev === 'desc' ? 'asc' : 'desc')
  }

  const totalPages = transactionData ? Math.ceil(transactionData.total / pageSize) : 0

  return (
    <div className="p-6">
      {/* Filters */}
      <div className="flex flex-wrap gap-4 mb-6">
        {/* Exchange Filter */}
        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm mb-1 text-nofx-text">
            {language === 'zh' ? '交易所' : 'Exchange'}
          </label>
          <select
            value={exchangeId}
            onChange={(e) => { setExchangeId(e.target.value); setPage(1); }}
            className="w-full bg-black/40 border border-white/10 rounded px-3 py-2 text-sm text-nofx-text-main focus:outline-none focus:border-nofx-gold/50"
          >
            <option value="all">{language === 'zh' ? '所有交易所' : 'All Exchanges'}</option>
            {exchanges.map(ex => (
              <option key={ex.id} value={ex.id}>{ex.account_name || ex.name}</option>
            ))}
          </select>
        </div>

        {/* Trader Info Display */}
        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm mb-1 text-nofx-text">
            {language === 'zh' ? '当前交易员' : 'Current Trader'}
          </label>
          <div className="w-full bg-black/40 border border-white/10 rounded px-3 py-2 text-sm text-nofx-text-main">
            {traderName || traderId}
          </div>
        </div>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="flex justify-center py-12">
          <Loader2 className="w-8 h-8 animate-spin text-nofx-gold" />
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="text-center py-12 text-red-500">
          {language === 'zh' ? '加载失败' : 'Failed to load'}
        </div>
      )}

      {/* Transaction Table */}
      {!isLoading && !error && transactionData && (
        <>
          <div className="overflow-x-auto mb-6">
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
                  <th className="px-4 py-3">{language === 'zh' ? '客户单号' : 'Client ID'}</th>
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
                {transactionData.items.map((tx: Transaction) => (
                  <tr key={tx.id} className="border-b border-white/5 hover:bg-white/5 transition-colors">
                    <td className="px-4 py-3 font-mono text-xs opacity-60">{tx.id}</td>
                    <td className="px-4 py-3 text-nofx-text-muted whitespace-nowrap">
                      {new Date(tx.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-nofx-text-muted" title={tx.exchange_order_id}>
                      {tx.exchange_order_id.length > 12 ? tx.exchange_order_id.slice(0, 8) + '...' : tx.exchange_order_id}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-nofx-text-muted">
                      {tx.client_order_id || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded bg-white/5 text-xs border border-white/10">
                         {exchanges.find(e => e.id === tx.exchange_id)?.account_name || tx.exchange_type}
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
                      <span className="px-2 py-0.5 rounded bg-white/5 text-xs text-nofx-text-muted">
                        {traderName || tx.trader_id || '-'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-xs text-nofx-text-muted mb-1">
                {language === 'zh' ? '总成交数' : 'Total Transactions'}
              </div>
              <div className="text-xl font-bold text-nofx-text">
                {transactionData.total}
              </div>
            </div>
            
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-xs text-nofx-text-muted mb-1">
                {language === 'zh' ? '盈利交易' : 'Profitable'}
              </div>
              <div className="text-xl font-bold text-nofx-green">
                {transactionData.items.filter((tx: Transaction) => tx.realized_pnl > 0).length}
              </div>
            </div>
            
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-xs text-nofx-text-muted mb-1">
                {language === 'zh' ? '亏损交易' : 'Loss-making'}
              </div>
              <div className="text-xl font-bold text-nofx-red">
                {transactionData.items.filter((tx: Transaction) => tx.realized_pnl < 0).length}
              </div>
            </div>
            
            <div className="bg-white/5 rounded-lg p-4 border border-white/10">
              <div className="text-xs text-nofx-text-muted mb-1">
                {language === 'zh' ? '净盈亏' : 'Net PnL'}
              </div>
              <div className={`text-xl font-bold ${transactionData.items.reduce((sum: number, tx: Transaction) => sum + (tx.realized_pnl || 0), 0) >= 0 ? 'text-nofx-green' : 'text-nofx-red'}`}>
                {transactionData.items.reduce((sum: number, tx: Transaction) => sum + (tx.realized_pnl || 0), 0) >= 0 ? '+' : ''}
                {transactionData.items.reduce((sum: number, tx: Transaction) => sum + (tx.realized_pnl || 0), 0).toFixed(2)}
              </div>
            </div>
          </div>

          {/* Pagination */}
          <div className="flex flex-wrap justify-between items-center text-sm text-nofx-text-muted gap-4">
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
                disabled={page === 1 || isValidating}
                className="px-3 py-1 rounded bg-white/5 hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {language === 'zh' ? '上一页' : 'Previous'}
              </button>
              <span className="px-3 py-1">{page} / {totalPages}</span>
              <button
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages || isValidating}
                className="px-3 py-1 rounded bg-white/5 hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {language === 'zh' ? '下一页' : 'Next'}
              </button>
            </div>
          </div>
        </>
      )}

      {/* Empty State */}
      {!isLoading && !error && transactionData && transactionData.items.length === 0 && (
        <div className="text-center py-12 text-nofx-text-muted">
          <div className="text-4xl mb-4">📊</div>
          <div className="text-lg font-semibold mb-2">
            {language === 'zh' ? '暂无成交记录' : 'No Transaction Records'}
          </div>
          <div>
            {language === 'zh' ? '调整筛选条件或等待更多交易数据' : 'Adjust filters or wait for more trading data'}
          </div>
        </div>
      )}
    </div>
  )
}