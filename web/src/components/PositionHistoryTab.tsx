import { useState, useEffect, useMemo } from 'react'
import { api } from '../lib/api'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { Loader2, TrendingUp, TrendingDown, BarChart3, Target, DollarSign } from 'lucide-react'
import type { TraderStats, HistoricalPosition, DirectionStats } from '../types'

interface PositionHistoryTabProps {
  traderId: string
  traderName: string
}

// Format number with proper decimals
function formatNumber(value: number, decimals: number = 2): string {
  if (Math.abs(value) >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  }
  if (Math.abs(value) >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  }
  return value.toFixed(decimals)
}

// Format duration from minutes
function formatDuration(minutes: number): string {
  if (!minutes || minutes <= 0) return '-'
  if (minutes < 60) return `${minutes.toFixed(0)}m`
  if (minutes < 1440) return `${(minutes / 60).toFixed(1)}h`
  return `${(minutes / 1440).toFixed(1)}d`
}

// Stat Card Component
function StatCard({
  title,
  value,
  suffix,
  color = '#EAECEF',
  icon,
  subtitle,
}: {
  title: string
  value: string | number
  suffix?: string
  color?: string
  icon: React.ReactNode
  subtitle?: string
}) {
  return (
    <div
      className="rounded-lg p-4 transition-all duration-200 hover:scale-[1.02]"
      style={{
        background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
        border: '1px solid #2B3139',
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.2)',
      }}
    >
      <div className="flex items-center gap-2 mb-2">
        {icon}
        <span className="text-xs" style={{ color: '#848E9C' }}>
          {title}
        </span>
      </div>
      <div className="flex items-baseline gap-1">
        <span
          className="text-xl font-bold font-mono"
          style={{ color }}
        >
          {value}
        </span>
        {suffix && (
          <span className="text-sm" style={{ color: '#848E9C' }}>
            {suffix}
          </span>
        )}
      </div>
      {subtitle && (
        <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
          {subtitle}
        </div>
      )}
    </div>
  )
}

export function PositionHistoryTab({ traderId, traderName }: PositionHistoryTabProps) {
  const { language } = useLanguage()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [positions, setPositions] = useState<HistoricalPosition[]>([])
  const [stats, setStats] = useState<TraderStats | null>(null)
  const [directionStats, setDirectionStats] = useState<DirectionStats[]>([])

  // Fetch position history when traderId changes
  useEffect(() => {
    const fetchData = async () => {
      if (!traderId) {
        setPositions([])
        setStats(null)
        setDirectionStats([])
        return
      }

      try {
        setLoading(true)
        setError(null)
        const data = await api.getPositionHistory(traderId, 200)
        setPositions(data.positions || [])
        setStats(data.stats)
        setDirectionStats(data.direction_stats || [])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load position history')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [traderId])

  // Calculate profit/loss ratio
  const profitLossRatio = useMemo(() => {
    if (!stats) return 0
    const avgWin = stats.avg_win || 0
    const avgLoss = stats.avg_loss || 0
    if (avgLoss === 0) return avgWin > 0 ? Infinity : 0
    return avgWin / avgLoss
  }, [stats])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-nofx-gold" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6 text-center text-red-500">
        {error}
      </div>
    )
  }

  return (
    <div className="p-6">
      {traderId && (
        <>
          {/* Statistics Overview */}
          {stats && (
            <div className="mb-8">
              <h2 className="text-lg font-semibold mb-4 text-nofx-text flex items-center gap-2">
                <BarChart3 className="w-5 h-5 text-nofx-gold" />
                {language === 'zh' ? '交易统计' : 'Trading Statistics'}
              </h2>
              
              {/* Core Metrics */}
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4 mb-4">
                <StatCard
                  icon={<BarChart3 className="w-4 h-4 text-blue-400" />}
                  title={t('positionHistory.totalTrades', language)}
                  value={stats.total_trades || 0}
                  subtitle={`${stats.win_trades || 0} ${language === 'zh' ? '盈利' : 'wins'} / ${stats.loss_trades || 0} ${language === 'zh' ? '亏损' : 'losses'}`}
                />
                <StatCard
                  icon={<Target className="w-4 h-4 text-green-400" />}
                  title={t('positionHistory.winRate', language)}
                  value={(stats.win_rate || 0).toFixed(1)}
                  suffix="%"
                  color={
                    (stats.win_rate || 0) >= 60
                      ? '#0ECB81'
                      : (stats.win_rate || 0) >= 40
                        ? '#F0B90B'
                        : '#F6465D'
                  }
                />
                <StatCard
                  icon={<DollarSign className="w-4 h-4 text-yellow-400" />}
                  title={t('positionHistory.totalPnL', language)}
                  value={((stats.total_pnl || 0) >= 0 ? '+' : '') + formatNumber(stats.total_pnl || 0)}
                  color={(stats.total_pnl || 0) >= 0 ? '#0ECB81' : '#F6465D'}
                  subtitle={`${t('positionHistory.fee', language)}: -${formatNumber(stats.total_fee || 0)}`}
                />
                <StatCard
                  icon={<TrendingUp className="w-4 h-4 text-purple-400" />}
                  title={t('positionHistory.profitFactor', language)}
                  value={(stats.profit_factor || 0).toFixed(2)}
                  color={(stats.profit_factor || 0) >= 1.5 ? '#0ECB81' : (stats.profit_factor || 0) >= 1 ? '#F0B90B' : '#F6465D'}
                  subtitle={t('positionHistory.profitFactorDesc', language)}
                />
                <StatCard
                  icon={<TrendingDown className="w-4 h-4 text-pink-400" />}
                  title={t('positionHistory.plRatio', language)}
                  value={profitLossRatio === Infinity ? '∞' : profitLossRatio.toFixed(2)}
                  color={profitLossRatio >= 1.5 ? '#0ECB81' : profitLossRatio >= 1 ? '#F0B90B' : '#F6465D'}
                  subtitle={t('positionHistory.plRatioDesc', language)}
                />
              </div>

              {/* Advanced Metrics */}
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                <StatCard
                  icon={<BarChart3 className="w-4 h-4 text-cyan-400" />}
                  title={t('positionHistory.sharpeRatio', language)}
                  value={(stats.sharpe_ratio || 0).toFixed(2)}
                  color={(stats.sharpe_ratio || 0) >= 1 ? '#0ECB81' : (stats.sharpe_ratio || 0) >= 0 ? '#F0B90B' : '#F6465D'}
                  subtitle={t('positionHistory.sharpeRatioDesc', language)}
                />
                <StatCard
                  icon={<TrendingDown className="w-4 h-4 text-red-400" />}
                  title={t('positionHistory.maxDrawdown', language)}
                  value={(stats.max_drawdown_pct || 0).toFixed(1)}
                  suffix="%"
                  color={(stats.max_drawdown_pct || 0) <= 10 ? '#0ECB81' : (stats.max_drawdown_pct || 0) <= 20 ? '#F0B90B' : '#F6465D'}
                />
                <StatCard
                  icon={<TrendingUp className="w-4 h-4 text-green-400" />}
                  title={t('positionHistory.avgWin', language)}
                  value={'+' + formatNumber(stats.avg_win || 0)}
                  color="#0ECB81"
                />
                <StatCard
                  icon={<TrendingDown className="w-4 h-4 text-red-400" />}
                  title={t('positionHistory.avgLoss', language)}
                  value={'-' + formatNumber(stats.avg_loss || 0)}
                  color="#F6465D"
                />
                <StatCard
                  icon={<DollarSign className="w-4 h-4 text-yellow-400" />}
                  title={t('positionHistory.netPnL', language)}
                  value={((stats.total_pnl || 0) - (stats.total_fee || 0) >= 0 ? '+' : '') + formatNumber((stats.total_pnl || 0) - (stats.total_fee || 0))}
                  color={(stats.total_pnl || 0) - (stats.total_fee || 0) >= 0 ? '#0ECB81' : '#F6465D'}
                  subtitle={t('positionHistory.netPnLDesc', language)}
                />
              </div>
            </div>
          )}

          {/* Direction Stats */}
          {directionStats.length > 0 && (
            <div className="mb-8">
              <h2 className="text-lg font-semibold mb-4 text-nofx-text">
                {language === 'zh' ? '方向统计' : 'Direction Statistics'}
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {directionStats.map((stat) => {
                  const isLong = (stat.side || '').toLowerCase() === 'long'
                  const iconColor = isLong ? '#0ECB81' : '#F6465D'
                  const totalPnl = stat.total_pnl || 0
                  const winRate = stat.win_rate || 0
                  const tradeCount = stat.trade_count || 0
                  const avgPnl = stat.avg_pnl || 0
                  const pnlColor = totalPnl >= 0 ? '#0ECB81' : '#F6465D'

                  return (
                    <div
                      key={stat.side}
                      className="rounded-lg p-4"
                      style={{
                        background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
                        border: `1px solid ${iconColor}33`,
                      }}
                    >
                      <div className="flex items-center gap-2 mb-3">
                        <span className="text-xl">{isLong ? '📈' : '📉'}</span>
                        <span className="font-bold uppercase" style={{ color: iconColor }}>
                          {stat.side || 'Unknown'}
                        </span>
                      </div>
                      <div className="grid grid-cols-4 gap-4">
                        <div>
                          <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                            {t('positionHistory.trades', language)}
                          </div>
                          <div className="font-mono font-semibold" style={{ color: '#EAECEF' }}>
                            {tradeCount}
                          </div>
                        </div>
                        <div>
                          <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                            {t('positionHistory.winRate', language)}
                          </div>
                          <div
                            className="font-mono font-semibold"
                            style={{
                              color: winRate >= 60 ? '#0ECB81' : winRate >= 40 ? '#F0B90B' : '#F6465D',
                            }}
                          >
                            {winRate.toFixed(1)}%
                          </div>
                        </div>
                        <div>
                          <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                            {t('positionHistory.totalPnL', language)}
                          </div>
                          <div className="font-mono font-semibold" style={{ color: pnlColor }}>
                            {totalPnl >= 0 ? '+' : ''}{formatNumber(totalPnl)}
                          </div>
                        </div>
                        <div>
                          <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                            {t('positionHistory.avgPnL', language)}
                          </div>
                          <div className="font-mono font-semibold" style={{ color: avgPnl >= 0 ? '#0ECB81' : '#F6465D' }}>
                            {avgPnl >= 0 ? '+' : ''}{formatNumber(avgPnl)}
                          </div>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* Position History Table */}
          {positions.length > 0 && (
            <div>
              <h2 className="text-lg font-semibold mb-4 text-nofx-text">
                {language === 'zh' ? '历史仓位记录' : 'Position History'}
              </h2>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ background: '#0B0E11' }}>
                      <th className="py-3 px-4 text-left text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.symbol', language)}
                      </th>
                      <th className="py-3 px-4 text-right text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.entry', language)}
                      </th>
                      <th className="py-3 px-4 text-right text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.exit', language)}
                      </th>
                      <th className="py-3 px-4 text-right text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.qty', language)}
                      </th>
                      <th className="py-3 px-4 text-right text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.pnl', language)}
                      </th>
                      <th className="py-3 px-4 text-center text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.duration', language)}
                      </th>
                      <th className="py-3 px-4 text-right text-xs font-semibold uppercase tracking-wider" style={{ color: '#848E9C' }}>
                        {t('positionHistory.closedAt', language)}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {positions.slice(0, 20).map((position) => {
                      const side = position.side || ''
                      const isLong = side.toUpperCase() === 'LONG'
                      const realizedPnl = position.realized_pnl || 0
                      const isProfitable = realizedPnl >= 0
                      const sideColor = isLong ? '#0ECB81' : '#F6465D'
                      const pnlColor = isProfitable ? '#0ECB81' : '#F6465D'

                      // Calculate holding time
                      const entryTime = position.entry_time ? new Date(position.entry_time).getTime() : 0
                      const exitTime = position.exit_time ? new Date(position.exit_time).getTime() : 0
                      const holdingMinutes = entryTime && exitTime && exitTime > entryTime ? (exitTime - entryTime) / 60000 : 0

                      return (
                        <tr
                          key={position.id}
                          className="transition-all duration-200 hover:bg-white/5"
                          style={{ borderBottom: '1px solid #2B3139' }}
                        >
                          <td className="py-3 px-4">
                            <div className="flex items-center gap-2">
                              <span className="font-mono font-semibold" style={{ color: '#EAECEF' }}>
                                {(position.symbol || '').replace('USDT', '')}
                              </span>
                              <span
                                className="px-2 py-0.5 rounded text-xs font-semibold uppercase"
                                style={{
                                  background: `${sideColor}22`,
                                  color: sideColor,
                                  border: `1px solid ${sideColor}44`,
                                }}
                              >
                                {side}
                              </span>
                            </div>
                          </td>
                          <td className="py-3 px-4 text-right font-mono" style={{ color: '#EAECEF' }}>
                            {(position.entry_price || 0).toFixed(4)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono" style={{ color: '#EAECEF' }}>
                            {(position.exit_price || 0).toFixed(4)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono" style={{ color: '#848E9C' }}>
                            {(position.quantity || 0).toFixed(4)}
                          </td>
                          <td className="py-3 px-4 text-right">
                            <div className="font-mono font-semibold" style={{ color: pnlColor }}>
                              {isProfitable ? '+' : ''}{formatNumber(realizedPnl)}
                            </div>
                          </td>
                          <td className="py-3 px-4 text-center text-sm" style={{ color: '#848E9C' }}>
                            {formatDuration(holdingMinutes)}
                          </td>
                          <td className="py-3 px-4 text-right text-xs" style={{ color: '#848E9C' }}>
                            {position.exit_time ? new Date(position.exit_time).toLocaleString() : '-'}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
              {positions.length > 20 && (
                <div className="mt-4 text-center text-sm text-nofx-text-muted">
                  {language === 'zh' 
                    ? `显示前20条记录，共${positions.length}条` 
                    : `Showing first 20 records of ${positions.length} total`}
                </div>
              )}
            </div>
          )}

          {positions.length === 0 && (
            <div className="text-center py-12 text-nofx-text-muted">
              {language === 'zh' ? `交易员 ${traderName} 暂无历史仓位记录` : `No position history for trader ${traderName}`}
            </div>
          )}
        </>
      )}
    </div>
  )
}