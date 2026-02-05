import { useState, useEffect, useCallback } from 'react'
import { addMonths, subMonths } from 'date-fns'
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis } from 'recharts'
import Header from '../components/Header'
import { api } from '../api'
import type { CategoryStat, CardStat, LedgerStatistics, LedgerSettings } from '../types'
import styles from './StatisticsV2.module.css'

const COLORS = ['#ff9500', '#10b981', '#3b82f6', '#ef4444', '#8b5cf6', '#ec4899', '#f59e0b', '#14b8a6']

type StatsTab = 'ledger' | 'card'

function getDefaultMonth() {
  // 현재 날짜를 기본값으로
  return new Date()
}

export default function StatisticsV2() {
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [tab, setTab] = useState<StatsTab>('ledger')
  const [ledgerStats, setLedgerStats] = useState<LedgerStatistics | null>(null)
  const [cardStats, setCardStats] = useState<{ by_category: CategoryStat[], by_card: CardStat[], total: number, total_last_month: number } | null>(null)
  const [settings, setSettings] = useState<LedgerSettings | null>(null)
  const [, setLoading] = useState(true)
  const [showInfo, setShowInfo] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      if (tab === 'ledger') {
        const [stats, settingsData] = await Promise.all([
          api.v2.getStatisticsLedger(year, month),
          api.v2.getSettings()
        ])
        setLedgerStats(stats)
        setSettings(settingsData)
      } else {
        const stats = await api.v2.getStatisticsCard(year, month)
        setCardStats(stats)
      }
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }, [currentMonth, tab])

  useEffect(() => {
    loadData()
  }, [loadData])

  const formatMoney = (amount: number) => amount.toLocaleString() + '원'

  const renderLedgerStats = () => {
    if (!ledgerStats) return null
    const { total_income, total_expense, balance, by_category, by_asset, by_income_category } = ledgerStats

    return (
      <>
        <div className={styles.summaryCard}>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>총 수입</span>
            <span className={`${styles.summaryAmount} ${styles.income}`}>+{formatMoney(total_income)}</span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>총 지출</span>
            <span className={`${styles.summaryAmount} ${styles.expense}`}>-{formatMoney(total_expense)}</span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>합계</span>
            <span className={`${styles.summaryAmount} ${styles.balance}`}>{formatMoney(balance)}</span>
          </div>
        </div>

        {by_category && by_category.length > 0 && (
          <div className={styles.section}>
            <h3 className={styles.sectionTitle}>카테고리별 지출</h3>
            <div className={styles.chartContainer}>
              <ResponsiveContainer width="100%" height={180}>
                <PieChart>
                  <Pie data={by_category} dataKey="total_amount" nameKey="category_name" cx="50%" cy="50%" innerRadius={45} outerRadius={75}>
                    {by_category.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className={styles.categoryList}>
              {by_category.map((cat, i) => (
                <div key={cat.category_name} className={styles.categoryItem}>
                  <div className={styles.categoryLeft}>
                    <span className={styles.categoryDot} style={{ background: COLORS[i % COLORS.length] }} />
                    <span className={styles.categoryName}>{cat.category_name}</span>
                    <span className={styles.categoryPercent}>
                      {total_expense > 0 ? Math.round((cat.total_amount / total_expense) * 100) : 0}%
                    </span>
                  </div>
                  <span className={styles.categoryAmount}>{formatMoney(cat.total_amount)}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {by_asset && by_asset.length > 0 && (
          <div className={styles.section}>
            <h3 className={styles.sectionTitle}>자산별 지출</h3>
            <div className={styles.cardList}>
              {by_asset.map(a => (
                <div key={a.asset_name} className={styles.cardItem}>
                  <span className={styles.cardName}>{a.asset_name}</span>
                  <span className={styles.cardAmount}>{formatMoney(a.total_amount)}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {by_income_category && by_income_category.length > 0 && (
          <div className={styles.section}>
            <h3 className={styles.sectionTitle}>수입 카테고리</h3>
            <div className={styles.cardList}>
              {by_income_category.map(c => (
                <div key={c.category_name} className={styles.cardItem}>
                  <span className={styles.cardName}>{c.category_name}</span>
                  <span className={styles.cardAmount} style={{ color: 'var(--income-color)' }}>+{formatMoney(c.total_amount)}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </>
    )
  }

  const renderCardStats = () => {
    if (!cardStats) return null
    const { by_category, by_card, total, total_last_month } = cardStats
    const diff = total - total_last_month
    const diffPercent = total_last_month > 0 ? Math.round((diff / total_last_month) * 100) : 0

    return (
      <>
        <div className={styles.summaryCard}>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>이번 달 카드 지출</span>
            <span className={`${styles.summaryAmount} ${styles.expense}`}>{formatMoney(total)}</span>
          </div>
          <div className={styles.comparison}>
            <span className={styles.prevLabel}>지난달 {formatMoney(total_last_month)}</span>
            <span className={`${styles.diffBadge} ${diff > 0 ? styles.increase : diff < 0 ? styles.decrease : ''}`}>
              {diff > 0 ? '▲' : diff < 0 ? '▼' : ''} {Math.abs(diffPercent)}%
            </span>
          </div>
        </div>

        {by_category && by_category.length > 0 && (
          <div className={styles.section}>
            <h3 className={styles.sectionTitle}>카테고리별</h3>
            <div className={styles.chartContainer}>
              <ResponsiveContainer width="100%" height={180}>
                <PieChart>
                  <Pie data={by_category} dataKey="total_amount" nameKey="category_name" cx="50%" cy="50%" innerRadius={45} outerRadius={75}>
                    {by_category.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className={styles.categoryList}>
              {by_category.map((cat, i) => (
                <div key={cat.category_name} className={styles.categoryItem}>
                  <div className={styles.categoryLeft}>
                    <span className={styles.categoryDot} style={{ background: COLORS[i % COLORS.length] }} />
                    <span className={styles.categoryName}>{cat.category_name}</span>
                  </div>
                  <span className={styles.categoryAmount}>{formatMoney(cat.total_amount)}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {by_card && by_card.length > 0 && (
          <div className={styles.section}>
            <h3 className={styles.sectionTitle}>카드별</h3>
            <div className={styles.chartContainerSmall}>
              <ResponsiveContainer width="100%" height={Math.max(by_card.length * 35, 70)}>
                <BarChart data={by_card} layout="vertical" margin={{ left: 0, right: 10 }}>
                  <XAxis type="number" hide />
                  <YAxis type="category" dataKey="card_name" width={80} tick={{ fontSize: 12, fill: 'var(--text-secondary)' }} />
                  <Bar dataKey="total_amount" fill="var(--accent-color)" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        )}
      </>
    )
  }

  return (
    <div className={styles.page}>
      <Header
        title="통계"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
        onInfoClick={() => setShowInfo(true)}
      />

      <div className={styles.tabBar}>
        <button className={`${styles.tab} ${tab === 'ledger' ? styles.activeTab : ''}`} onClick={() => setTab('ledger')}>
          가계부 통계
        </button>
        <button className={`${styles.tab} ${tab === 'card' ? styles.activeTab : ''}`} onClick={() => setTab('card')}>
          카드 통계
        </button>
      </div>

      {tab === 'ledger' ? renderLedgerStats() : renderCardStats()}

      {showInfo && (
        <>
          <div className={styles.overlay} onClick={() => setShowInfo(false)} />
          <div className={styles.infoModal}>
            <h3>📊 통계 기준 안내</h3>
            <p><strong>가계부 통계</strong></p>
            <div className={styles.highlight}>
              설정된 가계부 관리 기간 (매월 {settings?.ledger_period_start_day || 1}일~{settings?.ledger_period_end_day || 31}일) 기준으로 모든 수입/지출 통계를 산출합니다.
            </div>
            <p><strong>카드 통계</strong></p>
            <div className={styles.highlight}>
              각 카드별 이용 기간 및 결제 예정일을 기준으로 카드 지출 통계를 산출합니다.
            </div>
            <button onClick={() => setShowInfo(false)}>확인</button>
          </div>
        </>
      )}
    </div>
  )
}
