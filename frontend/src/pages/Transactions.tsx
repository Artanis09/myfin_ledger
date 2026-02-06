import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths } from 'date-fns'
import { Search, X, CreditCard, Target, Edit2, TrendingUp, TrendingDown, BarChart3, ChevronDown, ChevronUp } from 'lucide-react'
import Header from '../components/Header'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction, Card, Category, BillingPeriod } from '../types'
import styles from './Transactions.module.css'

type SortType = 'date-desc' | 'date-asc' | 'amount-desc' | 'amount-asc';
type TabType = 'history' | 'management';

interface WeekData {
  week_start: string
  week_end: string
  week_label: string
  total: number
}

function getDefaultMonth() {
  const now = new Date()
  return addMonths(now, 1)
}

export default function Transactions() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [cards, setCards] = useState<Card[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabType>('history')
  
  // 필터 상태
  const [showOnlyInstallment, setShowOnlyInstallment] = useState(false)
  const [sortBy, setSortBy] = useState<SortType>('date-desc')
  const [selectedCardId, setSelectedCardId] = useState<number | null>(null)
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [showSearch, setShowSearch] = useState(false)
  
  // 지출관리 탭 데이터
  const [billingPeriods, setBillingPeriods] = useState<BillingPeriod[]>([])
  const [weeklyStats, setWeeklyStats] = useState<WeekData[]>([])
  const [goal, setGoal] = useState<number | null>(null)
  const [showGoalInput, setShowGoalInput] = useState(false)
  const [goalInput, setGoalInput] = useState('')
  const [billingExpanded, setBillingExpanded] = useState<boolean>(() => {
    const saved = localStorage.getItem('billingExpanded')
    return saved !== 'false' // 기본값 true (펼쳐진 상태)
  })
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      const [txData, cardData, catData, dashData, weekData, goalData] = await Promise.all([
        api.getTransactions(year, month),
        api.getCards(),
        api.getCategories(),
        api.getDashboard(year, month),
        api.getWeeklyStats(year, month),
        api.getGoal(year, month),
      ])
      
      setTransactions(txData.transactions || [])
      setCards(cardData.cards || [])
      setCategories(catData.categories || [])
      setBillingPeriods(dashData.billing_periods || [])
      setWeeklyStats(weekData.weeks || [])
      setGoal(goalData.target_amount)
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }, [currentMonth])
  
  useEffect(() => {
    loadData()
  }, [loadData])
  
  useEffect(() => {
    const interval = setInterval(() => {
      loadData()
    }, 10000)
    return () => clearInterval(interval)
  }, [loadData])
  
  const handleEdit = (id: number) => {
    navigate(`/edit/${id}`)
  }
  
  const handleDelete = async (ids: number[]) => {
    try {
      await Promise.all(ids.map(id => api.deleteTransaction(id)))
      loadData()
    } catch (err) {
      console.error('Failed to delete:', err)
      alert('삭제에 실패했습니다.')
    }
  }

  const filteredAndSortedTransactions = useMemo(() => {
    let result = [...transactions];

    if (selectedCardId !== null) {
      result = result.filter(tx => tx.card_id === selectedCardId)
    }

    if (selectedCategoryId !== null) {
      result = result.filter(tx => tx.category_id === selectedCategoryId)
    }

    if (showOnlyInstallment) {
      result = result.filter(tx => tx.is_installment === 1);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase()
      result = result.filter(tx => 
        tx.description.toLowerCase().includes(query) ||
        (tx.memo && tx.memo.toLowerCase().includes(query)) ||
        (tx.card_name && tx.card_name.toLowerCase().includes(query)) ||
        (tx.category_name && tx.category_name.toLowerCase().includes(query))
      )
    }

    result.sort((a, b) => {
      switch (sortBy) {
        case 'date-desc':
          return new Date(b.transaction_date).getTime() - new Date(a.transaction_date).getTime();
        case 'date-asc':
          return new Date(a.transaction_date).getTime() - new Date(b.transaction_date).getTime();
        case 'amount-desc':
          return b.amount - a.amount;
        case 'amount-asc':
          return a.amount - b.amount;
        default:
          return 0;
      }
    });

    return result;
  }, [transactions, showOnlyInstallment, sortBy, selectedCardId, selectedCategoryId, searchQuery]);
  
  const totalExpense = filteredAndSortedTransactions.reduce((sum, tx) => {
    return sum + tx.amount
  }, 0)

  const clearFilters = () => {
    setSelectedCardId(null)
    setSelectedCategoryId(null)
    setShowOnlyInstallment(false)
    setSearchQuery('')
    setShowSearch(false)
  }

  const hasActiveFilters = selectedCardId !== null || selectedCategoryId !== null || showOnlyInstallment || searchQuery.trim()
  
  // 지출관리 탭 관련
  const totalBilling = billingPeriods.reduce((sum, bp) => sum + bp.total, 0)
  const goalProgress = goal ? Math.min((totalBilling / goal) * 100, 100) : 0
  const isOverBudget = goal ? totalBilling > goal : false
  const maxWeekTotal = Math.max(...weeklyStats.map(w => w.total), 1)
  
  const toggleBillingExpanded = () => {
    const newValue = !billingExpanded
    setBillingExpanded(newValue)
    localStorage.setItem('billingExpanded', String(newValue))
  }
  
  const handleSetGoal = async () => {
    const amount = parseInt(goalInput.replace(/,/g, ''), 10)
    if (isNaN(amount) || amount <= 0) {
      alert('올바른 금액을 입력하세요.')
      return
    }
    try {
      await api.setGoal(currentMonth.getFullYear(), currentMonth.getMonth() + 1, amount)
      setGoal(amount)
      setShowGoalInput(false)
      setGoalInput('')
    } catch (err) {
      alert('목표 설정에 실패했습니다.')
    }
  }
  
  return (
    <div className={styles.page}>
      <Header 
        title="카드관리"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      {/* 탭 선택 */}
      <div className={styles.tabBar}>
        <button 
          className={`${styles.tab} ${activeTab === 'history' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('history')}
        >
          내역
        </button>
        <button 
          className={`${styles.tab} ${activeTab === 'management' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('management')}
        >
          지출관리
        </button>
      </div>
      
      {activeTab === 'history' ? (
        <>
          {/* 검색 바 */}
          {showSearch && (
            <div className={styles.searchBar}>
              <Search size={18} className={styles.searchIcon} />
              <input
                type="text"
                placeholder="거래처, 내용, 메모 검색..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className={styles.searchInput}
                autoFocus
              />
              {searchQuery && (
                <button className={styles.clearSearchBtn} onClick={() => setSearchQuery('')}>
                  <X size={16} />
                </button>
              )}
            </div>
          )}
          
          {/* 필터 바 - 1행: 검색, 카드, 카테고리 */}
          <div className={styles.filterBar}>
            <div className={styles.filterRow}>
              <div className={styles.filterGroup}>
                <button 
                  className={`${styles.filterBtn} ${showSearch || searchQuery ? styles.active : ''}`}
                  onClick={() => setShowSearch(!showSearch)}
                >
                  <Search size={14} />
                </button>
                
                <select
                  className={`${styles.filterSelect} ${selectedCardId !== null ? styles.active : ''}`}
                  value={selectedCardId ?? ''}
                  onChange={e => setSelectedCardId(e.target.value ? Number(e.target.value) : null)}
                >
                  <option value="">전체 카드</option>
                  {cards.map(card => (
                    <option key={card.id} value={card.id}>{card.name}</option>
                  ))}
                </select>
                
                <select
                  className={`${styles.filterSelect} ${selectedCategoryId !== null ? styles.active : ''}`}
                  value={selectedCategoryId ?? ''}
                  onChange={e => setSelectedCategoryId(e.target.value ? Number(e.target.value) : null)}
                >
                  <option value="">전체 카테고리</option>
                  {categories.map(cat => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
                </select>
              </div>
              
              {hasActiveFilters && (
                <button className={styles.clearFilterBtn} onClick={clearFilters}>
                  초기화
                </button>
              )}
            </div>
            
            {/* 필터 바 - 2행: 할부, 정렬 */}
            <div className={styles.filterRow}>
              <div className={styles.filterGroup}>
                <button 
                  className={`${styles.filterBtn} ${showOnlyInstallment ? styles.active : ''}`}
                  onClick={() => setShowOnlyInstallment(!showOnlyInstallment)}
                >
                  할부만 보기
                </button>
              </div>
              
              <select 
                className={styles.sortSelect}
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as SortType)}
              >
                <option value="date-desc">날짜 최신순</option>
                <option value="date-asc">날짜 오래된순</option>
                <option value="amount-desc">금액 높은순</option>
                <option value="amount-asc">금액 낮은순</option>
              </select>
            </div>
          </div>
          
          {/* 필터 아래 구분선 */}
          <div className={styles.filterDivider} />

          {/* 지출 합계 */}
          <div className={styles.summary}>
            <span className={styles.totalAmount}>{totalExpense.toLocaleString()}원</span>
          </div>
          
          {filteredAndSortedTransactions.length > 0 ? (
            <TransactionList 
              transactions={filteredAndSortedTransactions}
              onEdit={handleEdit}
              onDelete={handleDelete}
              sortBy={sortBy}
              grouped={sortBy.startsWith('date')}
            />
          ) : (
            <div className={styles.empty}>
              <p>{hasActiveFilters ? '필터 결과가 없습니다' : '이 달 결제예정 내역이 없습니다'}</p>
            </div>
          )}
        </>
      ) : (
        <>
          {/* 지출관리 탭 */}
          {/* 예상 결제 금액 */}
          <div className={styles.billingSection}>
            <div className={styles.billingHeader} onClick={toggleBillingExpanded}>
              <div className={styles.billingLeft}>
                <div className={styles.billingIconWrapper}>
                  <CreditCard size={20} />
                </div>
                <div className={styles.billingHeaderInfo}>
                  <span className={styles.billingTitle}>예상 결제 금액</span>
                  <span className={styles.billingTotal}>{totalBilling.toLocaleString()}원</span>
                </div>
              </div>
              <div className={styles.billingToggle}>
                {billingExpanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
              </div>
            </div>
            
            {billingExpanded && billingPeriods.length > 0 && (
              <div className={styles.billingList}>
                {billingPeriods.map(bp => (
                  <div key={bp.card_id} className={styles.billingItem}>
                    <div className={styles.billingInfo}>
                      <span className={styles.billingCard}>{bp.card_name}</span>
                      <span className={styles.billingPeriod}>{bp.start_date}~{bp.end_date}</span>
                    </div>
                    <span className={styles.billingAmount}>
                      {bp.total.toLocaleString()}원
                    </span>
                  </div>
                ))}
              </div>
            )}
            
            {billingPeriods.length === 0 && (
              <div className={styles.noBilling}>
                <p>이 달 결제 예정 내역이 없습니다</p>
              </div>
            )}
          </div>
          
          {/* 월 소비 목표 */}
          <div className={styles.goalSection}>
            <div className={styles.goalHeader}>
              <div className={styles.goalTitle}>
                <div className={styles.iconWrapper}>
                  <Target size={18} />
                </div>
                <span>월 소비 목표</span>
              </div>
              <button onClick={() => setShowGoalInput(true)} className={styles.editGoalBtn}>
                <Edit2 size={14} />
              </button>
            </div>
            
            {showGoalInput ? (
              <div className={styles.goalInputBox}>
                <input
                  type="text"
                  placeholder="목표 금액 (원)"
                  value={goalInput}
                  onChange={(e) => setGoalInput(e.target.value.replace(/[^0-9]/g, '').replace(/\B(?=(\d{3})+(?!\d))/g, ','))}
                  className={styles.goalInput}
                />
                <button onClick={handleSetGoal} className={styles.goalSaveBtn}>저장</button>
                <button onClick={() => setShowGoalInput(false)} className={styles.goalCancelBtn}>취소</button>
              </div>
            ) : goal ? (
              <div className={styles.goalProgress}>
                <div className={styles.goalBar}>
                  <div 
                    className={`${styles.goalFill} ${isOverBudget ? styles.overBudget : ''}`}
                    style={{ width: `${goalProgress}%` }}
                  />
                </div>
                <div className={styles.goalInfo}>
                  <span className={styles.goalCurrent}>
                    {totalBilling.toLocaleString()}원
                    {isOverBudget && <TrendingUp size={14} className={styles.overIcon} />}
                  </span>
                  <span className={styles.goalTarget}>/ {goal.toLocaleString()}원</span>
                </div>
                <div className={styles.goalRemaining}>
                  {isOverBudget 
                    ? <span className={styles.over}>{(totalBilling - goal).toLocaleString()}원 초과</span>
                    : <span className={styles.under}>{(goal - totalBilling).toLocaleString()}원 남음</span>
                  }
                </div>
              </div>
            ) : (
              <div className={styles.noGoal}>
                <button onClick={() => setShowGoalInput(true)} className={styles.setGoalBtn}>
                  목표 설정하기
                </button>
              </div>
            )}
          </div>
          
          {/* 주간 이용금액 차트 */}
          {weeklyStats.length > 0 && (
            <div className={styles.weeklySection}>
              <h3 className={styles.sectionTitle}>
                <div className={styles.iconWrapper}>
                  <BarChart3 size={18} />
                </div>
                주간 이용 현황
              </h3>
              <div className={styles.weeklyChart}>
                {weeklyStats.map((week, idx) => {
                  const prevWeek = weeklyStats[idx - 1]
                  const diff = prevWeek ? week.total - prevWeek.total : 0
                  const heightPercent = maxWeekTotal > 0 ? (week.total / maxWeekTotal) * 100 : 0
                  const barClass = idx === 0 ? 'neutral' : (diff > 0 ? 'up' : diff < 0 ? 'down' : 'neutral')
                  
                  return (
                    <div key={week.week_start} className={styles.weekBar}>
                      <div className={styles.barContainer}>
                        <div 
                          className={`${styles.bar} ${styles[barClass]}`}
                          style={{ height: `${Math.max(heightPercent, 8)}%` }}
                        />
                      </div>
                      <div className={styles.weekAmount}>
                        {week.total > 0 ? `${(week.total / 10000).toFixed(0)}만` : '0'}
                      </div>
                      <div className={styles.weekLabel}>{week.week_label.split('~')[0]}</div>
                      {diff !== 0 && idx > 0 && (
                        <div className={`${styles.weekDiff} ${diff > 0 ? styles.up : styles.down}`}>
                          {diff > 0 ? <TrendingUp size={10} /> : <TrendingDown size={10} />}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </>
      )}
      
      {activeTab === 'history' && <FloatingButton />}
    </div>
  )
}
