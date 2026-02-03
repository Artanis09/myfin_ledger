import { useState, useEffect, useCallback } from 'react'
import { addMonths, subMonths } from 'date-fns'
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis } from 'recharts'
import Header from '../components/Header'
import { api } from '../api'
import type { CategoryStat, CardStat, Card } from '../types'
import styles from './Statistics.module.css'

const COLORS = ['#ff6b6b', '#4dabf7', '#69db7c', '#ffd43b', '#da77f2', '#ff922b', '#20c997', '#748ffc']

// 결제예정월 기준으로 필터링
function getBillingPeriodForMonth(year: number, month: number, startDay: number, endDay: number) {
  if (startDay <= endDay) {
    const prevMonth = month - 1 === 0 ? 12 : month - 1
    const prevYear = month - 1 === 0 ? year - 1 : year
    return {
      start: new Date(prevYear, prevMonth - 1, startDay),
      end: new Date(prevYear, prevMonth - 1, endDay, 23, 59, 59)
    }
  } else {
    const startMonth = month - 2 <= 0 ? month - 2 + 12 : month - 2
    const startYear = month - 2 <= 0 ? year - 1 : year
    const endMonth = month - 1 === 0 ? 12 : month - 1
    const endYear = month - 1 === 0 ? year - 1 : year
    return {
      start: new Date(startYear, startMonth - 1, startDay),
      end: new Date(endYear, endMonth - 1, endDay, 23, 59, 59)
    }
  }
}

// 매달 1일이면 다음달(결제예정월)을 기본으로 표시
function getDefaultMonth() {
  const now = new Date()
  return addMonths(now, 1)
}

export default function Statistics() {
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [byCategory, setByCategory] = useState<CategoryStat[]>([])
  const [byCard, setByCard] = useState<CardStat[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      // 카드 정보 로드
      const cardData = await api.getCards()
      const cardList = cardData.cards || []
      
      // 결제예정월 기준으로 통계 계산
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      // 모든 거래 로드
      const txData = await api.getTransactions()
      const allTransactions = txData.transactions || []
      
      // 카드별 이용기간 맵 생성
      const cardBillingMap = new Map<number, { startDay: number, endDay: number }>()
      cardList.forEach((card: Card) => {
        cardBillingMap.set(card.id, {
          startDay: card.billing_start_day,
          endDay: card.billing_end_day
        })
      })
      
      // 결제예정월 기준으로 필터링
      interface Transaction {
        card_id: number
        card_name: string
        category_name: string | null
        amount: number
        transaction_date: string
        is_cancelled: number
      }
      
      const filtered = allTransactions.filter((tx: Transaction) => {
        const billing = cardBillingMap.get(tx.card_id)
        if (!billing) return false
        
        const { start, end } = getBillingPeriodForMonth(
          year, month, billing.startDay, billing.endDay
        )
        
        const txDate = new Date(tx.transaction_date)
        return txDate >= start && txDate <= end
      })
      
      // 카테고리별 통계
      const categoryStats: Record<string, { total: number, count: number }> = {}
      filtered.forEach((tx: Transaction) => {
        const catName = tx.category_name || '미분류'
        if (!categoryStats[catName]) {
          categoryStats[catName] = { total: 0, count: 0 }
        }
        // 취소 거래는 마이너스
        const amount = tx.is_cancelled === 1 ? -tx.amount : tx.amount
        categoryStats[catName].total += amount
        categoryStats[catName].count += 1
      })
      
      const catList = Object.entries(categoryStats)
        .map(([name, stat]) => ({
          category_name: name,
          total_amount: stat.total,
          count: stat.count
        }))
        .filter(c => c.total_amount > 0)
        .sort((a, b) => b.total_amount - a.total_amount)
      
      setByCategory(catList)
      
      // 카드별 통계
      const cardStats: Record<number, { name: string, total: number, count: number }> = {}
      filtered.forEach((tx: Transaction) => {
        if (!cardStats[tx.card_id]) {
          cardStats[tx.card_id] = { name: tx.card_name, total: 0, count: 0 }
        }
        const amount = tx.is_cancelled === 1 ? -tx.amount : tx.amount
        cardStats[tx.card_id].total += amount
        cardStats[tx.card_id].count += 1
      })
      
      const cardStatList = Object.entries(cardStats)
        .map(([id, stat]) => ({
          card_id: parseInt(id),
          card_name: stat.name,
          total_amount: stat.total,
          count: stat.count
        }))
        .filter(c => c.total_amount > 0)
        .sort((a, b) => b.total_amount - a.total_amount)
      
      setByCard(cardStatList)
      
      // 총액
      const totalAmount = filtered.reduce((sum: number, tx: Transaction) => {
        return sum + (tx.is_cancelled === 1 ? -tx.amount : tx.amount)
      }, 0)
      setTotal(totalAmount)
      
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }, [currentMonth])
  
  useEffect(() => {
    loadData()
  }, [loadData])
  
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  
  return (
    <div className={styles.page}>
      <Header 
        title="통계"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      <div className={styles.totalSection}>
        <span className={styles.totalLabel}>지출</span>
        <span className={styles.totalAmount}>{formatMoney(total)}</span>
      </div>
      
      {byCategory.length > 0 && (
        <div className={styles.section}>
          <h3 className={styles.sectionTitle}>카테고리별 지출</h3>
          <div className={styles.chartContainer}>
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie
                  data={byCategory}
                  dataKey="total_amount"
                  nameKey="category_name"
                  cx="50%"
                  cy="50%"
                  innerRadius={50}
                  outerRadius={80}
                >
                  {byCategory.map((_, index) => (
                    <Cell key={index} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
          </div>
          <div className={styles.categoryList}>
            {byCategory.map((cat, index) => (
              <div key={cat.category_name} className={styles.categoryItem}>
                <div className={styles.categoryLeft}>
                  <span 
                    className={styles.categoryDot} 
                    style={{ background: COLORS[index % COLORS.length] }}
                  />
                  <span className={styles.categoryName}>{cat.category_name}</span>
                  <span className={styles.categoryPercent}>
                    {total > 0 ? Math.round((cat.total_amount / total) * 100) : 0}%
                  </span>
                </div>
                <span className={styles.categoryAmount}>{formatMoney(cat.total_amount)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {byCard.length > 0 && (
        <div className={styles.section}>
          <h3 className={styles.sectionTitle}>카드별 지출</h3>
          <div className={styles.chartContainerSmall}>
            <ResponsiveContainer width="100%" height={Math.max(byCard.length * 36, 80)}>
              <BarChart data={byCard} layout="vertical" margin={{ left: 0, right: 10 }}>
                <XAxis type="number" hide />
                <YAxis type="category" dataKey="card_name" width={80} tick={{ fontSize: 12 }} />
                <Bar dataKey="total_amount" fill="var(--accent-color)" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
          <div className={styles.cardList}>
            {byCard.map(card => (
              <div key={card.card_id} className={styles.cardItem}>
                <span className={styles.cardName}>{card.card_name}</span>
                <span className={styles.cardAmount}>{formatMoney(card.total_amount)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {byCategory.length === 0 && byCard.length === 0 && !loading && (
        <div className={styles.empty}>
          <p>이번 달 거래내역이 없습니다</p>
        </div>
      )}
    </div>
  )
}
