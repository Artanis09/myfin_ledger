import { useState, useRef, type TouchEvent } from 'react'
import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import { Trash2 } from 'lucide-react'
import styles from './TransactionList.module.css'
import type { Transaction } from '../types'

interface TransactionListProps {
  transactions: Transaction[]
  onEdit?: (id: number) => void
  onDelete?: (id: number) => void
  grouped?: boolean
}

export default function TransactionList({ transactions, onEdit, onDelete, grouped = true }: TransactionListProps) {
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  
  // Group by date
  const groupedTx = transactions.reduce((acc, tx) => {
    const date = tx.transaction_date.split('T')[0]
    if (!acc[date]) acc[date] = []
    acc[date].push(tx)
    return acc
  }, {} as Record<string, Transaction[]>)
  
  const sortedDates = Object.keys(groupedTx).sort((a, b) => b.localeCompare(a))
  
  const getDayTotal = (txs: Transaction[]) => {
    return txs.reduce((sum, tx) => sum + tx.amount, 0)
  }
  
  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return {
      day: date.getDate(),
      dayOfWeek: format(date, 'EEEE', { locale: ko }),
      full: format(date, 'yyyy.MM', { locale: ko }),
    }
  }
  
  if (!grouped) {
    return (
      <div className={styles.list}>
        {transactions.map(tx => (
          <TransactionItem 
            key={tx.id} 
            tx={tx} 
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
      </div>
    )
  }
  
  return (
    <div className={styles.container}>
      {sortedDates.map(date => {
        const { day, dayOfWeek, full } = formatDate(date)
        const dayTxs = groupedTx[date]
        const dayTotal = getDayTotal(dayTxs)
        
        return (
          <div key={date} className={styles.dayGroup}>
            <div className={styles.dayHeader}>
              <div className={styles.dayInfo}>
                <span className={styles.dayNum}>{day}</span>
                <div className={styles.dayMeta}>
                  <span className={styles.dayDate}>{full}</span>
                  <span className={styles.dayOfWeek}>{dayOfWeek}</span>
                </div>
              </div>
              <div className={styles.dayTotals}>
                <span className={styles.dayIncome}>0원</span>
                <span className={styles.dayExpense}>{formatMoney(dayTotal)}</span>
              </div>
            </div>
            <div className={styles.list}>
              {dayTxs.map(tx => (
                <TransactionItem 
                  key={tx.id} 
                  tx={tx}
                  onEdit={onEdit}
                  onDelete={onDelete}
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function TransactionItem({ tx, onEdit, onDelete }: { 
  tx: Transaction
  onEdit?: (id: number) => void
  onDelete?: (id: number) => void
}) {
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  const time = tx.transaction_date.split('T')[1]?.slice(0, 5) || ''
  
  const [swipeOffset, setSwipeOffset] = useState(0)
  const [isSwiping, setIsSwiping] = useState(false)
  const startX = useRef(0)
  const startY = useRef(0)
  const isHorizontalSwipe = useRef<boolean | null>(null)
  
  const DELETE_THRESHOLD = 80
  
  const handleTouchStart = (e: TouchEvent) => {
    startX.current = e.touches[0].clientX
    startY.current = e.touches[0].clientY
    isHorizontalSwipe.current = null
    setIsSwiping(true)
  }
  
  const handleTouchMove = (e: TouchEvent) => {
    if (!isSwiping) return
    
    const currentX = e.touches[0].clientX
    const currentY = e.touches[0].clientY
    const diffX = startX.current - currentX
    const diffY = startY.current - currentY
    
    // Determine swipe direction on first significant movement
    if (isHorizontalSwipe.current === null && (Math.abs(diffX) > 5 || Math.abs(diffY) > 5)) {
      isHorizontalSwipe.current = Math.abs(diffX) > Math.abs(diffY)
    }
    
    // Only handle horizontal swipes
    if (isHorizontalSwipe.current) {
      e.preventDefault()
      // Only allow left swipe (positive diffX)
      const offset = Math.max(0, Math.min(diffX, DELETE_THRESHOLD + 20))
      setSwipeOffset(offset)
    }
  }
  
  const handleTouchEnd = () => {
    setIsSwiping(false)
    if (swipeOffset > DELETE_THRESHOLD) {
      // Trigger delete
      setSwipeOffset(DELETE_THRESHOLD)
      if (onDelete) {
        onDelete(tx.id)
      }
    } else {
      setSwipeOffset(0)
    }
    isHorizontalSwipe.current = null
  }
  
  const handleClick = () => {
    if (swipeOffset > 0) {
      setSwipeOffset(0)
    } else {
      onEdit?.(tx.id)
    }
  }
  
  return (
    <div className={styles.swipeContainer}>
      <div 
        className={styles.deleteAction}
        style={{ width: swipeOffset }}
        onClick={() => onDelete?.(tx.id)}
      >
        <Trash2 size={20} />
      </div>
      <div 
        className={styles.item}
        style={{ transform: `translateX(-${swipeOffset}px)` }}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        onClick={handleClick}
      >
        <div className={styles.itemLeft}>
          <span className={styles.category}>{tx.category_name || '기타'}</span>
          <div className={styles.itemInfo}>
            <span className={styles.description}>{tx.description}</span>
            <span className={styles.meta}>
              {time && `${time} · `}{tx.card_name}
            </span>
          </div>
        </div>
        <div className={styles.itemRight}>
          <span className={styles.amount}>{formatMoney(tx.amount)}</span>
          {tx.is_installment === 1 && (
            <span className={styles.installment}>
              {tx.installment_current}/{tx.installment_months}개월
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
