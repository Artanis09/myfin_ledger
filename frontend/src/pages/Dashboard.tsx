import { useState, useEffect } from 'react'
import { addMonths, subMonths, startOfMonth, endOfMonth, format } from 'date-fns'
import Header from '../components/Header'
import MonthSummary from '../components/MonthSummary'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction, BillingPeriod } from '../types'
import styles from './Dashboard.module.css'

export default function Dashboard() {
  const [currentMonth, setCurrentMonth] = useState(new Date())
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [billingPeriods, setBillingPeriods] = useState<BillingPeriod[]>([])
  const [totalExpense, setTotalExpense] = useState(0)
  const [, setLoading] = useState(true)
  
  useEffect(() => {
    loadData()
  }, [currentMonth])
  
  const loadData = async () => {
    setLoading(true)
    try {
      const start = format(startOfMonth(currentMonth), 'yyyy-MM-dd')
      const end = format(endOfMonth(currentMonth), 'yyyy-MM-dd')
      
      const [dashData, txData] = await Promise.all([
        api.getDashboard(),
        api.getStatistics(start, end),
      ])
      
      setTransactions(dashData.recent_transactions || [])
      setBillingPeriods(dashData.billing_periods || [])
      setTotalExpense(txData.total || 0)
    } catch (err) {
      console.error('Failed to load data:', err)
    } finally {
      setLoading(false)
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
      
      <MonthSummary income={0} expense={totalExpense} />
      
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
      
      <div className={styles.transactionsSection}>
        <h3 className={styles.sectionTitle}>최근 거래</h3>
        {transactions.length > 0 ? (
          <TransactionList transactions={transactions.slice(0, 10)} />
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
