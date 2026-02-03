import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths } from 'date-fns'
import Header from '../components/Header'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction, Card } from '../types'
import styles from './Transactions.module.css'

// 결제예정월 기준으로 필터링
// 예: 카드 이용기간 23~22일 경우, 1월 23일 ~ 2월 22일 사용내역은 3월 결제예정
function getBillingPeriodForMonth(year: number, month: number, startDay: number, endDay: number) {
  // month는 1-12 (결제예정월)
  // 결제예정월의 이전 달에 해당하는 이용기간을 계산
  
  if (startDay <= endDay) {
    // 예: 1~31 (같은 달 내)
    // 이전 달의 startDay ~ endDay
    const prevMonth = month - 1 === 0 ? 12 : month - 1
    const prevYear = month - 1 === 0 ? year - 1 : year
    return {
      start: new Date(prevYear, prevMonth - 1, startDay),
      end: new Date(prevYear, prevMonth - 1, endDay, 23, 59, 59)
    }
  } else {
    // 예: 23~22 (두 달에 걸침)
    // 이전이전 달 startDay ~ 이전 달 endDay
    // 3월 결제 -> 1월 23일 ~ 2월 22일
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

export default function Transactions() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(new Date())
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [, setCards] = useState<Card[]>([])
  const [, setLoading] = useState(true)
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [txData, cardData] = await Promise.all([
        api.getTransactions(),
        api.getCards()
      ])
      
      const allTransactions = txData.transactions || []
      const cardList = cardData.cards || []
      setCards(cardList)
      
      // 카드별 이용기간 맵 생성
      const cardBillingMap = new Map<number, { startDay: number, endDay: number }>()
      cardList.forEach((card: Card) => {
        cardBillingMap.set(card.id, {
          startDay: card.billing_start_day,
          endDay: card.billing_end_day
        })
      })
      
      // 현재 선택된 월(결제예정월) 기준으로 필터링
      const billingYear = currentMonth.getFullYear()
      const billingMonth = currentMonth.getMonth() + 1 // 1-12
      
      const filtered = allTransactions.filter((tx: Transaction) => {
        const billing = cardBillingMap.get(tx.card_id)
        if (!billing) return false
        
        const { start, end } = getBillingPeriodForMonth(
          billingYear, billingMonth, billing.startDay, billing.endDay
        )
        
        const txDate = new Date(tx.transaction_date)
        return txDate >= start && txDate <= end
      })
      
      setTransactions(filtered)
    } catch (err) {
      console.error('Failed to load:', err)
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
          <p>이 달 결제예정 내역이 없습니다</p>
        </div>
      )}
      
      <FloatingButton />
    </div>
  )
}
