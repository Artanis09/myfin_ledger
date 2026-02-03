import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths, startOfMonth, endOfMonth, format } from 'date-fns'
import { Target, TrendingUp, TrendingDown, Edit2 } from 'lucide-react'
import Header from '../components/Header'
import MonthSummary from '../components/MonthSummary'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction, BillingPeriod } from '../types'
import styles from './Dashboard.module.css'

interface WeekData {
  week_start: string
  week_end: string
  week_label: string
  total: number
  by_card: { card_id: number; card_name: string; total: number }[]
}

export default function Dashboard() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(new Date())
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [billingPeriods, setBillingPeriods] = useState<BillingPeriod[]>([])
  const [, setTotalExpense] = useState(0)
  const [weeklyStats, setWeeklyStats] = useState<WeekData[]>([])
  const [goal, setGoal] = useState<number | null>(null)
  const [showGoalInput, setShowGoalInput] = useState(false)
  const [goalInput, setGoalInput] = useState('')
  const [, setLoading] = useState(true)
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      const start = format(startOfMonth(currentMonth), 'yyyy-MM-dd')
      const end = format(endOfMonth(currentMonth), 'yyyy-MM-dd')
      
      const [dashData, txData, weekData, goalData] = await Promise.all([
        api.getDashboard(),
        api.getStatistics(start, end),
        api.getWeeklyStats(year, month),
        api.getGoal(year, month),
      ])
      
      setTransactions(dashData.recent_transactions || [])
      setBillingPeriods(dashData.billing_periods || [])
      setTotalExpense(txData.total || 0)
      setWeeklyStats(weekData.weeks || [])
      setGoal(goalData.target_amount)
    } catch (err) {
      console.error('Failed to load data:', err)
    } finally {
      setLoading(false)
    }
  }, [currentMonth])
  
  useEffect(() => {
    loadData()
  }, [loadData])
  
  // Auto refresh every 10 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      loadData()
    }, 10000)
    return () => clearInterval(interval)
  }, [loadData])
  
  const handleEdit = (id: number) => {
    navigate(`/transactions/${id}/edit`)
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
  
  // Calculate total from billing periods (결제예정금액 기준)
  const totalBilling = billingPeriods.reduce((sum, bp) => sum + bp.total, 0)
  const goalProgress = goal ? Math.min((totalBilling / goal) * 100, 100) : 0
  const isOverBudget = goal ? totalBilling > goal : false
  
  // Find max week total for chart scaling
  const maxWeekTotal = Math.max(...weeklyStats.map(w => w.total), 1)
  
  return (
    <div className={styles.page}>
      <Header 
        title="가계부"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      <MonthSummary income={0} expense={totalBilling} />
      
      {/* 월 소비 목표 */}
      <div className={styles.goalSection}>
        <div className={styles.goalHeader}>
          <div className={styles.goalTitle}>
            <Target size={18} />
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
          <h3 className={styles.sectionTitle}>주간 이용 현황</h3>
          <div className={styles.weeklyChart}>
            {weeklyStats.map((week, idx) => {
              const prevWeek = weeklyStats[idx - 1]
              const diff = prevWeek ? week.total - prevWeek.total : 0
              const heightPercent = (week.total / maxWeekTotal) * 100
              
              return (
                <div key={week.week_start} className={styles.weekBar}>
                  <div className={styles.barContainer}>
                    <div 
                      className={styles.bar}
                      style={{ height: `${Math.max(heightPercent, 5)}%` }}
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
      
      {/* 예상 결제 금액 */}
      {billingPeriods.length > 0 && (
        <div className={styles.billingSection}>
          <h3 className={styles.sectionTitle}>예상 결제 금액</h3>
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
        </div>
      )}
      
      {/* 최근 거래 */}
      <div className={styles.transactionsSection}>
        <h3 className={styles.sectionTitle}>최근 거래</h3>
        {transactions.length > 0 ? (
          <TransactionList 
            transactions={transactions.slice(0, 10)} 
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
        ) : (
          <div className={styles.empty}>
            <p>등록된 거래가 없습니다</p>
          </div>
        )}
      </div>
      
      <FloatingButton />
    </div>
  )
}
