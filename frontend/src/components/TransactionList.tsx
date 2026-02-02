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
  
  const [isOpen, setIsOpen] = useState(false)
  const [translateX, setTranslateX] = useState(0)
  const [isDragging, setIsDragging] = useState(false)
  const startX = useRef(0)
  const startY = useRef(0)
  const isHorizontal = useRef<boolean | null>(null)
  
  const DELETE_BTN_WIDTH = 70
  
  const handleTouchStart = (e: TouchEvent) => {
    startX.current = e.touches[0].clientX
    startY.current = e.touches[0].clientY
    setIsDragging(true)
    isHorizontal.current = null
  }
  
  const handleTouchMove = (e: TouchEvent) => {
    if (!isDragging) return
    
    const touchX = e.touches[0].clientX
    const touchY = e.touches[0].clientY
    const diffX = startX.current - touchX
    const diffY = startY.current - touchY
    
    // Determine direction on first move
    if (isHorizontal.current === null) {
      if (Math.abs(diffX) > 10 || Math.abs(diffY) > 10) {
        isHorizontal.current = Math.abs(diffX) > Math.abs(diffY)
      }
      return
    }
    
    if (!isHorizontal.current) return
    
    // Calculate new position
    let newX: number
    if (isOpen) {
      newX = DELETE_BTN_WIDTH - diffX
    } else {
      newX = diffX
    }
    
    newX = Math.max(0, Math.min(newX, DELETE_BTN_WIDTH))
    setTranslateX(newX)
  }
  
  const handleTouchEnd = () => {
    setIsDragging(false)
    isHorizontal.current = null
    
    if (translateX > DELETE_BTN_WIDTH / 2) {
      setIsOpen(true)
      setTranslateX(DELETE_BTN_WIDTH)
    } else {
      setIsOpen(false)
      setTranslateX(0)
    }
  }
  
  const handleItemClick = () => {
    if (isOpen) {
      setIsOpen(false)
      setTranslateX(0)
    } else if (translateX === 0) {
      onEdit?.(tx.id)
    }
  }
  
  const handleDeleteClick = () => {
    if (onDelete) {
      onDelete(tx.id)
    }
  }
  
  const offset = isDragging ? translateX : (isOpen ? DELETE_BTN_WIDTH : 0)
  
  return (
    <div className={styles.swipeContainer}>
      <div 
        className={styles.deleteAction}
        onTouchEnd={(e) => { e.stopPropagation(); handleDeleteClick(); }}
        onClick={(e) => { e.stopPropagation(); handleDeleteClick(); }}
      >
        <Trash2 size={20} />
        <span>삭제</span>
      </div>
      <div 
        className={`${styles.item} ${isDragging ? '' : styles.itemAnimated}`}
        style={{ transform: `translateX(-${offset}px)` }}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        onClick={handleItemClick}
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
