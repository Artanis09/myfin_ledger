import { useState, useEffect } from 'react'
import { addMonths, subMonths, startOfMonth, endOfMonth, format } from 'date-fns'
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip } from 'recharts'
import Header from '../components/Header'
import { api } from '../api'
import type { CategoryStat, CardStat } from '../types'
import styles from './Statistics.module.css'

const COLORS = ['#ff6b6b', '#4dabf7', '#69db7c', '#ffd43b', '#da77f2', '#ff922b', '#20c997', '#748ffc']

export default function Statistics() {
  const [currentMonth, setCurrentMonth] = useState(new Date())
  const [byCategory, setByCategory] = useState<CategoryStat[]>([])
  const [byCard, setByCard] = useState<CardStat[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  
  useEffect(() => {
    loadData()
  }, [currentMonth])
  
  const loadData = async () => {
    setLoading(true)
    try {
      const start = format(startOfMonth(currentMonth), 'yyyy-MM-dd')
      const end = format(endOfMonth(currentMonth), 'yyyy-MM-dd')
      const data = await api.getStatistics(start, end)
      
      setByCategory(data.by_category || [])
      setByCard(data.by_card || [])
      setTotal(data.total || 0)
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }
  
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
          <div className={styles.chartContainer}>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={byCard} layout="vertical">
                <XAxis type="number" hide />
                <YAxis type="category" dataKey="card_name" width={80} tick={{ fontSize: 12 }} />
                <Tooltip formatter={(value) => formatMoney(Number(value))} />
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
