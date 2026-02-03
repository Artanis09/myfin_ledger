import { useState, useEffect, useCallback } from 'react'
import { addMonths, subMonths } from 'date-fns'
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis } from 'recharts'
import Header from '../components/Header'
import { api } from '../api'
import type { CategoryStat, CardStat } from '../types'
import styles from './Statistics.module.css'

const COLORS = ['#ff9f0a', '#8e8e93', '#ff6b8a', '#64d2ff', '#bf5af2', '#30d158', '#ff453a', '#ffd60a']

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
  const [prevTotal, setPrevTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      const stats = await api.getStatistics(year, month)
      
      setByCategory(stats.by_category || [])
      setByCard(stats.by_card || [])
      setTotal(stats.total || 0)
      setPrevTotal(stats.total_last_month || 0)
      
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
  
  const diff = total - prevTotal
  const diffPercent = prevTotal > 0 ? Math.round((diff / prevTotal) * 100) : (total > 0 ? 100 : 0)
  
  return (
    <div className={styles.page}>
      <Header 
        title="통계"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      <div className={styles.totalSection}>
        <span className={styles.totalLabel}>이번 달 지출</span>
        <span className={styles.totalAmount}>{formatMoney(total)}</span>
        <div className={styles.comparison}>
          <span className={styles.prevAmount}>지난달 {formatMoney(prevTotal)}</span>
          <span className={`${styles.diffBadge} ${diff > 0 ? styles.increase : diff < 0 ? styles.decrease : ''}`}>
            {diff > 0 ? '▲' : diff < 0 ? '▼' : ''} {Math.abs(diffPercent)}%
          </span>
        </div>
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
            <ResponsiveContainer width="100%" height={Math.max(byCard.length * 30, 60)}>
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
