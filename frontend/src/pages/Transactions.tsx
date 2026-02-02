import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths, startOfMonth, endOfMonth } from 'date-fns'
import Header from '../components/Header'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction } from '../types'
import styles from './Transactions.module.css'

export default function Transactions() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(new Date())
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [, setLoading] = useState(true)
  
  useEffect(() => {
    loadData()
  }, [currentMonth])
  
  const loadData = async () => {
    setLoading(true)
    try {
      const data = await api.getTransactions()
      
      // Filter by month
      const start = startOfMonth(currentMonth)
      const end = endOfMonth(currentMonth)
      const filtered = (data.transactions || []).filter((tx: Transaction) => {
        const txDate = new Date(tx.transaction_date)
        return txDate >= start && txDate <= end
      })
      
      setTransactions(filtered)
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }
  
  const handleEdit = (id: number) => {
    navigate(`/transactions/${id}/edit`)
  }
  
  const handleDelete = async (id: number) => {
    try {
      await api.deleteTransaction(id)
      loadData()
    } catch (err) {
      console.error('Failed to delete:', err)
      alert('삭제에 실패했습니다.')
    }
  }
  
  const totalExpense = transactions.reduce((sum, tx) => sum + tx.amount, 0)
  
  return (
    <div className={styles.page}>
      <Header 
        title="거래내역"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      <div className={styles.summary}>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>수입</span>
          <span className={styles.summaryIncome}>0원</span>
        </div>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>지출</span>
          <span className={styles.summaryExpense}>{totalExpense.toLocaleString()}원</span>
        </div>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>합계</span>
          <span className={styles.summaryTotal}>-{totalExpense.toLocaleString()}원</span>
        </div>
      </div>
      
      {transactions.length > 0 ? (
        <TransactionList 
          transactions={transactions}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      ) : (
        <div className={styles.empty}>
          <p>이번 달 거래내역이 없습니다</p>
        </div>
      )}
      
      <FloatingButton />
    </div>
  )
}
