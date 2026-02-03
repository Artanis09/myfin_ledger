import { useState, useRef } from 'react'
import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import { Trash2 } from 'lucide-react'
import styles from './TransactionList.module.css'
import type { Transaction } from '../types'

interface TransactionListProps {
  transactions: Transaction[]
  onEdit?: (id: number) => void
  onDelete?: (ids: number[]) => void
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
  
  const handleDelete = async (id: number) => {
    onDelete?.([id])
  }
  
  if (!grouped) {
    return (
      <div className={styles.list}>
        {transactions.map(tx => (
          <SwipeableItem 
            key={tx.id} 
            tx={tx} 
            onEdit={() => onEdit?.(tx.id)}
            onDelete={() => handleDelete(tx.id)}
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
                <SwipeableItem 
                  key={tx.id} 
                  tx={tx}
                  onEdit={() => onEdit?.(tx.id)}
                  onDelete={() => handleDelete(tx.id)}
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function SwipeableItem({ tx, onEdit, onDelete }: { 
  tx: Transaction
  onEdit: () => void
  onDelete: () => void
}) {
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  const time = tx.transaction_date.split('T')[1]?.slice(0, 5) || ''
  
  const [swiped, setSwiped] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const touchStartX = useRef(0)
  const touchCurrentX = useRef(0)
  const [translateX, setTranslateX] = useState(0)
  const itemRef = useRef<HTMLDivElement>(null)
  
  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
    touchCurrentX.current = e.touches[0].clientX
  }
  
  const handleTouchMove = (e: React.TouchEvent) => {
    touchCurrentX.current = e.touches[0].clientX
    const diff = touchCurrentX.current - touchStartX.current
    
    // Only allow left swipe (negative diff)
    if (diff < 0) {
      const newTranslate = Math.max(diff, -80)
      setTranslateX(newTranslate)
    } else if (swiped) {
      // If already swiped, allow closing
      const newTranslate = Math.min(diff - 80, 0)
      setTranslateX(newTranslate)
    }
  }
  
  const handleTouchEnd = () => {
    const diff = touchCurrentX.current - touchStartX.current
    
    if (diff < -40) {
      // Swipe left threshold reached - show delete button
      setTranslateX(-80)
      setSwiped(true)
    } else if (diff > 40 && swiped) {
      // Swipe right to close
      setTranslateX(0)
      setSwiped(false)
    } else {
      // Reset to current state
      setTranslateX(swiped ? -80 : 0)
    }
  }
  
  const handleClick = () => {
    if (swiped) {
      setTranslateX(0)
      setSwiped(false)
    } else {
      onEdit()
    }
  }
  
  const handleDeleteClick = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (isDeleting) return
    
    setIsDeleting(true)
    try {
      await onDelete()
    } finally {
      setIsDeleting(false)
    }
  }
  
  return (
    <div className={styles.swipeWrapper}>
      <div className={styles.deleteAction}>
        <button 
          className={styles.deleteBtn}
          onClick={handleDeleteClick}
          disabled={isDeleting}
        >
          <Trash2 size={20} />
          <span>삭제</span>
        </button>
      </div>
      <div 
        ref={itemRef}
        className={styles.item}
        style={{ transform: `translateX(${translateX}px)` }}
        onClick={handleClick}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        <div className={styles.itemLeft}>
          <span className={styles.category}>
            {tx.is_cancelled === 1 && <span className={styles.cancelledBadge}>취소</span>}
            {tx.category_name || '기타'}
          </span>
          <div className={styles.itemInfo}>
            <span className={`${styles.description} ${tx.is_cancelled === 1 ? styles.cancelled : ''}`}>
              {tx.description}
            </span>
            <span className={styles.meta}>
              {time && `${time} · `}{tx.card_name}
            </span>
          </div>
        </div>
        <div className={styles.itemRight}>
          <span className={`${styles.amount} ${tx.is_cancelled === 1 ? styles.cancelledAmount : ''}`}>
            {tx.is_cancelled === 1 ? '-' : ''}{formatMoney(tx.amount)}
          </span>
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
