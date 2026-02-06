import { useState, useRef } from 'react'
import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import { Trash2 } from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'
import styles from './TransactionList.module.css'
import type { Transaction } from '../types'

interface TransactionListProps {
  transactions: Transaction[]
  onEdit?: (id: number) => void
  onDelete?: (ids: number[]) => void
  grouped?: boolean
  sortBy?: string
}

export default function TransactionList({ transactions, onEdit, onDelete, grouped = true, sortBy = 'date-desc' }: TransactionListProps) {
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  
  // Group by date while preserving the order from parent (for date-based sorting)
  // For amount-based sorting, we don't group
  const isDateSort = sortBy.startsWith('date');
  
  // Group by date - preserve order of first appearance
  const groupedTx: Record<string, Transaction[]> = {};
  const dateOrder: string[] = [];
  
  transactions.forEach(tx => {
    const date = tx.transaction_date.split('T')[0];
    if (!groupedTx[date]) {
      groupedTx[date] = [];
      dateOrder.push(date);
    }
    groupedTx[date].push(tx);
  });
  
  // For date sorting, use the preserved order; for amount sorting, sort dates by first tx amount
  const sortedDates = isDateSort 
    ? dateOrder  // Already in correct order from parent
    : dateOrder.sort((a, b) => {
        const amountA = groupedTx[a][0]?.amount || 0;
        const amountB = groupedTx[b][0]?.amount || 0;
        return sortBy === 'amount-desc' ? amountB - amountA : amountA - amountB;
      })
  
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
  const { showMemo } = useTheme()
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  const time = tx.transaction_date.split('T')[1]?.slice(0, 5) || ''
  const isCard = !!tx.card_name
  const assetName = tx.card_name || '현금'
  
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
          <div className={`${styles.categoryIcon} ${styles.expense}`}>
            {(tx.category_name || '기타').slice(0, 4)}
          </div>
          <div className={styles.itemInfo}>
            <div className={styles.descriptionRow}>
              {tx.is_cancelled === 1 && <span className={styles.cancelledBadge}>취소</span>}
              <span className={`${styles.description} ${tx.is_cancelled === 1 ? styles.cancelled : ''}`}>
                {tx.description}
              </span>
            </div>
            <div className={styles.meta}>
              <span className={`${styles.assetName} ${isCard ? styles.card : styles.other}`}>{assetName}</span>
              <span className={styles.dot}>•</span>
              <span className={styles.time}>{time}</span>
            </div>
            {showMemo && tx.memo && (
              <div className={styles.memo}>{tx.memo}</div>
            )}
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
