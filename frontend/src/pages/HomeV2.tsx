import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths, format, startOfMonth, endOfMonth, eachDayOfInterval, getDay, isToday } from 'date-fns'
import { ko } from 'date-fns/locale'
import { Plus, Minus, ListIcon, Calendar, BarChart3 } from 'lucide-react'
import { PieChart, Pie, Cell, ResponsiveContainer } from 'recharts'
import Header from '../components/Header'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { TransactionV2, DashboardV2Data, LedgerSettings } from '../types'
import styles from './HomeV2.module.css'

const COLORS = ['#ff9500', '#10b981', '#3b82f6', '#ef4444', '#8b5cf6', '#ec4899', '#f59e0b', '#14b8a6']

type ViewMode = 'daily' | 'calendar' | 'monthly'
type TabType = 'history' | 'statistics'

// 통계 탭 컴포넌트
function StatisticsTab({ data, formatAmount }: { data: DashboardV2Data | null, formatAmount: (n: number) => string }) {
  const categoryData = useMemo(() => {
    if (!data?.transactions) return []
    const groups: Record<string, number> = {}
    data.transactions
      .filter(tx => tx.tx_type === 'expense' && !tx.is_cancelled)
      .forEach(tx => {
        const cat = tx.category_name || '미분류'
        groups[cat] = (groups[cat] || 0) + tx.amount
      })
    return Object.entries(groups)
      .sort((a, b) => b[1] - a[1])
      .map(([name, value]) => ({ name, value }))
  }, [data])

  const assetData = useMemo(() => {
    if (!data?.transactions) return []
    const groups: Record<string, number> = {}
    data.transactions
      .filter(tx => tx.tx_type === 'expense' && !tx.is_cancelled)
      .forEach(tx => {
        const asset = tx.asset_type_name || '미분류'
        groups[asset] = (groups[asset] || 0) + tx.amount
      })
    return Object.entries(groups)
      .sort((a, b) => b[1] - a[1])
      .map(([name, value]) => ({ name, value }))
  }, [data])

  const incomeData = useMemo(() => {
    if (!data?.transactions) return []
    const groups: Record<string, number> = {}
    data.transactions
      .filter(tx => tx.tx_type === 'income' && !tx.is_cancelled)
      .forEach(tx => {
        const cat = tx.income_category_name || '미분류'
        groups[cat] = (groups[cat] || 0) + tx.amount
      })
    return Object.entries(groups)
      .sort((a, b) => b[1] - a[1])
      .map(([name, value]) => ({ name, value }))
  }, [data])

  const totalExpense = categoryData.reduce((sum, item) => sum + item.value, 0)
  const totalIncome = incomeData.reduce((sum, item) => sum + item.value, 0)

  if (!data?.transactions?.length) {
    return <div className={styles.empty}><BarChart3 size={32} /><p>통계 데이터가 없습니다</p></div>
  }

  return (
    <div className={styles.statisticsContent}>
      {/* 카테고리별 지출 */}
      {categoryData.length > 0 && (
        <div className={styles.statSection}>
          <h3 className={styles.statTitle}>카테고리별 지출</h3>
          <div className={styles.chartWrapper}>
            <ResponsiveContainer width="100%" height={160}>
              <PieChart>
                <Pie
                  data={categoryData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  innerRadius={40}
                  outerRadius={70}
                >
                  {categoryData.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
          </div>
          <div className={styles.statList}>
            {categoryData.map((item, i) => (
              <div key={item.name} className={styles.statItem}>
                <div className={styles.statLeft}>
                  <span className={styles.statDot} style={{ background: COLORS[i % COLORS.length] }} />
                  <span className={styles.statName}>{item.name}</span>
                  <span className={styles.statPercent}>
                    {totalExpense > 0 ? Math.round((item.value / totalExpense) * 100) : 0}%
                  </span>
                </div>
                <span className={styles.statAmount}>{formatAmount(item.value)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 자산별 지출 */}
      {assetData.length > 0 && (
        <div className={styles.statSection}>
          <h3 className={styles.statTitle}>자산별 지출</h3>
          <div className={styles.statList}>
            {assetData.map((item, i) => {
              const percent = totalExpense > 0 ? (item.value / totalExpense) * 100 : 0
              return (
                <div key={item.name} className={styles.statBarItem}>
                  <div className={styles.statBarHeader}>
                    <span className={styles.statName}>{item.name}</span>
                    <span className={styles.statAmount}>{formatAmount(item.value)}</span>
                  </div>
                  <div className={styles.statBar}>
                    <div 
                      className={styles.statBarFill} 
                      style={{ width: `${percent}%`, background: COLORS[i % COLORS.length] }}
                    />
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* 수입 카테고리 */}
      {incomeData.length > 0 && (
        <div className={styles.statSection}>
          <h3 className={styles.statTitle}>수입 카테고리</h3>
          <div className={styles.statList}>
            {incomeData.map((item) => (
              <div key={item.name} className={styles.statItem}>
                <div className={styles.statLeft}>
                  <span className={styles.statName}>{item.name}</span>
                  <span className={styles.statPercent}>
                    {totalIncome > 0 ? Math.round((item.value / totalIncome) * 100) : 0}%
                  </span>
                </div>
                <span className={`${styles.statAmount} ${styles.incomeAmount}`}>+{formatAmount(item.value)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function getDefaultMonth() {
  // 현재 날짜를 기본값으로 (다음 달이 아닌 이번 달)
  return new Date()
}

export default function HomeV2() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [activeTab, setActiveTab] = useState<TabType>('history')
  const [viewMode, setViewMode] = useState<ViewMode>('daily')
  const [data, setData] = useState<DashboardV2Data | null>(null)
  const [settings, setSettings] = useState<LedgerSettings | null>(null)
  const [, setLoading] = useState(true)
  const [showInfo, setShowInfo] = useState(false)
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [yearSelectMode, setYearSelectMode] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      const [dashData, settingsData] = await Promise.all([
        api.v2.getDashboard(year, month),
        api.v2.getSettings()
      ])
      setData(dashData)
      setSettings(settingsData)
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }, [currentMonth])

  useEffect(() => {
    loadData()
  }, [loadData])

  const formatAmount = (amount: number) => {
    return new Intl.NumberFormat('ko-KR').format(amount) + '원'
  }

  const groupByDate = (txns: TransactionV2[]) => {
    const groups: Record<string, TransactionV2[]> = {}
    txns.forEach(tx => {
      const date = tx.transaction_date.substring(0, 10)
      if (!groups[date]) groups[date] = []
      groups[date].push(tx)
    })
    return Object.entries(groups).sort((a, b) => b[0].localeCompare(a[0]))
  }

  const getDayTotals = (txns: TransactionV2[]) => {
    let income = 0, expense = 0
    txns.forEach(tx => {
      if (tx.tx_type === 'income') income += tx.amount
      else expense += tx.amount
    })
    return { income, expense }
  }

  // 카테고리 첫 글자 추출
  const getCategoryInitial = (name: string | null | undefined) => {
    if (!name) return '미'
    return name.charAt(0)
  }

  const renderDailyView = () => {
    if (!data?.transactions?.length) {
      return <div className={styles.empty}><ListIcon size={32} /><p>등록된 내역이 없습니다</p></div>
    }
    const grouped = groupByDate(data.transactions)
    return (
      <div className={styles.transactionList}>
        {grouped.map(([date, txns]) => {
          const totals = getDayTotals(txns)
          return (
            <div key={date} className={styles.dayGroup}>
              <div className={styles.dayHeader}>
                <span>{format(new Date(date), 'M월 d일 (E)', { locale: ko })}</span>
                <div className={styles.dayTotals}>
                  {totals.income > 0 && <span className="income">+{formatAmount(totals.income)}</span>}
                  {totals.expense > 0 && <span className="expense">-{formatAmount(totals.expense)}</span>}
                </div>
              </div>
              {txns.map(tx => {
                const catName = tx.tx_type === 'income' ? tx.income_category_name : tx.category_name
                return (
                  <div key={tx.id} className={styles.txItem} onClick={() => navigate(`/transactions/${tx.id}/edit`)}>
                    <div className={`${styles.categoryBadge} ${tx.tx_type === 'income' ? styles.incomeBadge : styles.expenseBadge}`}>
                      {getCategoryInitial(catName)}
                    </div>
                    <div className={styles.txInfo}>
                      <div className={styles.txDesc}>{tx.description}</div>
                      <div className={styles.txMeta}>
                        {catName || '미분류'} · {tx.asset_type_name}
                      </div>
                    </div>
                    <div className={`${styles.txAmount} ${tx.tx_type === 'income' ? styles.income : styles.expense}`}>
                      {tx.tx_type === 'income' ? '+' : ''}{formatAmount(tx.amount)}
                    </div>
                  </div>
                )
              })}
            </div>
          )
        })}
      </div>
    )
  }

  const renderCalendarView = () => {
    if (!data) return null
    const start = startOfMonth(currentMonth)
    const end = endOfMonth(currentMonth)
    const days = eachDayOfInterval({ start, end })
    const startDayOfWeek = getDay(start)
    const emptyCells = Array(startDayOfWeek).fill(null)

    return (
      <div className={styles.calendar}>
        <div className={styles.calendarHeader}>
          {['일', '월', '화', '수', '목', '금', '토'].map(d => <div key={d}>{d}</div>)}
        </div>
        <div className={styles.calendarGrid}>
          {emptyCells.map((_, i) => <div key={`e${i}`} className={`${styles.calendarDay} ${styles.empty}`} />)}
          {days.map(day => {
            const dateStr = format(day, 'yyyy-MM-dd')
            const dayData = data.daily_summary[dateStr]
            const hasData = dayData && (dayData.income || dayData.expense)
            return (
              <div
                key={dateStr}
                className={`${styles.calendarDay} ${isToday(day) ? styles.today : ''} ${hasData ? styles.hasData : ''}`}
                onClick={() => hasData && setSelectedDate(dateStr)}
              >
                <span className={styles.dayNum}>{format(day, 'd')}</span>
                {dayData?.income && <span className={styles.dayIncome}>+{(dayData.income / 10000).toFixed(0)}</span>}
                {dayData?.expense && <span className={styles.dayExpense}>-{(dayData.expense / 10000).toFixed(0)}</span>}
              </div>
            )
          })}
        </div>
      </div>
    )
  }

  const renderMonthlyView = () => {
    const year = currentMonth.getFullYear()
    const months = Array.from({ length: 12 }, (_, i) => i + 1)
    return (
      <div className={styles.monthlyList}>
        {months.map(m => (
          <div key={m} className={styles.monthItem} onClick={() => {
            setCurrentMonth(new Date(year, m - 1, 1))
            setViewMode('daily')
            setYearSelectMode(false)
          }}>
            <span className={styles.monthName}>{m}월</span>
            <div className={styles.monthTotals}>
              <div className="income">+0원</div>
              <div className="expense">-0원</div>
              <div className="balance">0원</div>
            </div>
          </div>
        ))}
      </div>
    )
  }

  const getSelectedDateTxns = () => {
    if (!selectedDate || !data?.transactions) return []
    return data.transactions.filter(tx => tx.transaction_date.substring(0, 10) === selectedDate)
  }

  return (
    <div className={styles.page}>
      <Header
        title="가계부"
        month={yearSelectMode ? undefined : currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
        onInfoClick={() => setShowInfo(true)}
      />

      {/* 상단 탭 바 - 카드관리 디자인과 동일 */}
      <div className={styles.tabBar}>
        <button 
          className={`${styles.tab} ${activeTab === 'history' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('history')}
        >
          내역
        </button>
        <button 
          className={`${styles.tab} ${activeTab === 'statistics' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('statistics')}
        >
          통계
        </button>
      </div>

      <div className={styles.summary}>
        <div className={styles.summaryItem}>
          <div className={styles.summaryLabel}>총 수입</div>
          <div className={`${styles.summaryAmount} ${styles.income}`}>
            {data ? formatAmount(data.total_income) : '-'}
          </div>
        </div>
        <div className={styles.summaryItem}>
          <div className={styles.summaryLabel}>총 지출</div>
          <div className={`${styles.summaryAmount} ${styles.expense}`}>
            {data ? formatAmount(data.total_expense) : '-'}
          </div>
        </div>
        <div className={styles.summaryItem}>
          <div className={styles.summaryLabel}>합계</div>
          <div className={`${styles.summaryAmount} ${styles.balance}`}>
            {data ? formatAmount(data.balance) : '-'}
          </div>
        </div>
      </div>

      {activeTab === 'history' && (
        <div className={styles.viewTabs}>
          <button 
            className={`${styles.viewTab} ${viewMode === 'daily' ? styles.active : ''}`}
            onClick={() => { setViewMode('daily'); setYearSelectMode(false) }}
          >
            <ListIcon size={16} />일일
          </button>
          <button 
            className={`${styles.viewTab} ${viewMode === 'calendar' ? styles.active : ''}`}
            onClick={() => { setViewMode('calendar'); setYearSelectMode(false) }}
          >
            <Calendar size={16} />달력
          </button>
          <button 
            className={`${styles.viewTab} ${viewMode === 'monthly' ? styles.active : ''}`}
            onClick={() => { setViewMode('monthly'); setYearSelectMode(true) }}
          >
            <BarChart3 size={16} />월별
          </button>
        </div>
      )}

      <div className={styles.content}>
        {activeTab === 'history' && (
          <>
            {viewMode === 'daily' && renderDailyView()}
            {viewMode === 'calendar' && renderCalendarView()}
            {viewMode === 'monthly' && renderMonthlyView()}
          </>
        )}
        {activeTab === 'statistics' && <StatisticsTab data={data} formatAmount={formatAmount} />}
      </div>

      <div className={`${styles.overlay} ${selectedDate ? styles.open : ''}`} onClick={() => setSelectedDate(null)} />
      <div className={`${styles.bottomSheet} ${selectedDate ? styles.open : ''}`}>
        <div className={styles.sheetHandle} />
        <div className={styles.sheetHeader}>
          <span className={styles.sheetTitle}>
            {selectedDate && format(new Date(selectedDate), 'M월 d일 (E)', { locale: ko })}
          </span>
          <button className={styles.sheetClose} onClick={() => setSelectedDate(null)}>×</button>
        </div>
        <div className={styles.sheetContent}>
          {getSelectedDateTxns().map(tx => (
            <div key={tx.id} className={styles.txItem} onClick={() => navigate(`/transactions/${tx.id}/edit`)}>
              <div className={`${styles.txIcon} ${tx.tx_type === 'income' ? styles.income : styles.expense}`}>
                {tx.tx_type === 'income' ? <Plus size={18} /> : <Minus size={18} />}
              </div>
              <div className={styles.txInfo}>
                <div className={styles.txDesc}>{tx.description}</div>
                <div className={styles.txMeta}>
                  {tx.tx_type === 'income' ? tx.income_category_name : tx.category_name}
                </div>
              </div>
              <div className={`${styles.txAmount} ${tx.tx_type === 'income' ? styles.income : styles.expense}`}>
                {tx.tx_type === 'income' ? '+' : '-'}{formatAmount(tx.amount)}
              </div>
            </div>
          ))}
        </div>
      </div>

      {showInfo && (
        <>
          <div className={`${styles.overlay} ${styles.open}`} onClick={() => setShowInfo(false)} />
          <div className={styles.infoModal}>
            <h3>📊 관리 기준 안내</h3>
            <p><strong>가계부 관리 기간</strong></p>
            <div className={styles.highlight}>
              매월 {settings?.ledger_period_start_day || 1}일 ~ {settings?.ledger_period_end_day || 31}일
            </div>
            <p style={{ marginTop: 12 }}>
              모든 수입/지출 내역은 위 기간을 기준으로 통계됩니다.
            </p>
            <p><strong>카드 결제 기준</strong></p>
            <p>카드별 이용 기간 및 결제 예정일은 카드 설정에서 개별 관리됩니다.</p>
            <button onClick={() => setShowInfo(false)}>확인</button>
          </div>
        </>
      )}

      <FloatingButton />
    </div>
  )
}
