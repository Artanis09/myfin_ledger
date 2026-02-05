import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths } from 'date-fns'
import { Clock } from 'lucide-react'
import Header from '../components/Header'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction } from '../types'
import styles from './Dashboard.module.css'

// 매달 1일이면 다음달(결제예정월)을 기본으로 표시
function getDefaultMonth() {
  const now = new Date()
  return addMonths(now, 1)
}

export default function Dashboard() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [, setLoading] = useState(true)
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      const dashData = await api.getDashboard(year, month)
      setTransactions(dashData.recent_transactions || [])
    } catch (err) {
      console.error('Failed to load data:', err)
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
  
  return (
    <div className={styles.page}>
      <Header 
        title="가계부"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      {/* 최근 거래 */}
      <div className={styles.transactionsSection}>
        <h3 className={styles.sectionTitle}>
          <div className={styles.iconWrapper}>
            <Clock size={18} />
          </div>
          최근 거래
        </h3>
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
