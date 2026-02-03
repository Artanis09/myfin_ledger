import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths } from 'date-fns'
import Header from '../components/Header'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction, Card } from '../types'
import styles from './Transactions.module.css'

type SortType = 'date-desc' | 'date-asc' | 'amount-desc' | 'amount-asc';

// 매달 1일이면 다음달(결제예정월)을 기본으로 표시
function getDefaultMonth() {
  const now = new Date()
  return addMonths(now, 1)
}

export default function Transactions() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [, setCards] = useState<Card[]>([])
  const [, setLoading] = useState(true)
  const [showOnlyInstallment, setShowOnlyInstallment] = useState(false)
  const [sortBy, setSortBy] = useState<SortType>('date-desc')
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      const [txData, cardData] = await Promise.all([
        api.getTransactions(year, month),
        api.getCards()
      ])
      
      setTransactions(txData.transactions || [])
      setCards(cardData.cards || [])
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

  const filteredAndSortedTransactions = useMemo(() => {
    let result = [...transactions];

    if (showOnlyInstallment) {
      result = result.filter(tx => tx.is_installment === 1);
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
  }, [transactions, showOnlyInstallment, sortBy]);
  
  const totalExpense = filteredAndSortedTransactions.reduce((sum, tx) => {
    // 취소 거래는 마이너스로 계산
    if (tx.is_cancelled === 1) {
      return sum - tx.amount
    }
    return sum + tx.amount
  }, 0)
  
  return (
    <div className={styles.page}>
      <Header 
        title="거래내역"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      <div className={styles.filterBar}>
        <div className={styles.filterGroup}>
          <button 
            className={`${styles.filterBtn} ${showOnlyInstallment ? styles.active : ''}`}
            onClick={() => setShowOnlyInstallment(!showOnlyInstallment)}
          >
            할부만 보기
          </button>
        </div>
        
        <div className={styles.sortGroup}>
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
          <p>{showOnlyInstallment ? '이 달 할부 결제 내역이 없습니다' : '이 달 결제예정 내역이 없습니다'}</p>
        </div>
      )}
      
      <FloatingButton />
    </div>
  )
}
