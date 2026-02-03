import { useState } from 'react'
import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import { Trash2, Check } from 'lucide-react'
import styles from './TransactionList.module.css'
import type { Transaction } from '../types'

interface TransactionListProps {
  transactions: Transaction[]
  onEdit?: (id: number) => void
  onDelete?: (ids: number[]) => void
  grouped?: boolean
}

export default function TransactionList({ transactions, onEdit, onDelete, grouped = true }: TransactionListProps) {
  const [selectMode, setSelectMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  
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
  
  const toggleSelect = (id: number) => {
    const newSet = new Set(selectedIds)
    if (newSet.has(id)) {
      newSet.delete(id)
    } else {
      newSet.add(id)
    }
    setSelectedIds(newSet)
  }
  
  const handleDelete = () => {
    if (selectedIds.size === 0) return
    onDelete?.([...selectedIds])
    setSelectedIds(new Set())
    setSelectMode(false)
  }
  
  const cancelSelect = () => {
    setSelectedIds(new Set())
    setSelectMode(false)
  }
  
  const selectAll = () => {
    setSelectedIds(new Set(transactions.map(tx => tx.id)))
  }
  
  const handleItemClick = (tx: Transaction) => {
    if (selectMode) {
      toggleSelect(tx.id)
    } else {
      onEdit?.(tx.id)
    }
  }
  
  const handleItemLongPress = (id: number) => {
    setSelectMode(true)
    setSelectedIds(new Set([id]))
  }
  
  if (!grouped) {
    return (
      <div className={styles.list}>
        {transactions.map(tx => (
          <TransactionItem 
            key={tx.id} 
            tx={tx} 
            onClick={() => handleItemClick(tx)}
            onLongPress={() => handleItemLongPress(tx.id)}
            selectMode={selectMode}
            selected={selectedIds.has(tx.id)}
          />
        ))}
      </div>
    )
  }
  
  return (
    <div className={styles.container}>
      {selectMode && (
        <div className={styles.selectBar}>
          <button onClick={cancelSelect} className={styles.cancelBtn}>취소</button>
          <span className={styles.selectCount}>{selectedIds.size}건 선택</span>
          <div className={styles.selectActions}>
            <button onClick={selectAll} className={styles.selectAllBtn}>전체</button>
            <button onClick={handleDelete} className={styles.deleteBtn} disabled={selectedIds.size === 0}>
              <Trash2 size={18} /> 삭제
            </button>
          </div>
        </div>
      )}
      
      {!selectMode && onDelete && (
        <div className={styles.editBar}>
          <button onClick={() => setSelectMode(true)} className={styles.editBtn}>선택</button>
        </div>
      )}
      
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
                  onClick={() => handleItemClick(tx)}
                  onLongPress={() => handleItemLongPress(tx.id)}
                  selectMode={selectMode}
                  selected={selectedIds.has(tx.id)}
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function TransactionItem({ tx, onClick, onLongPress, selectMode, selected }: { 
  tx: Transaction
  onClick: () => void
  onLongPress: () => void
  selectMode: boolean
  selected: boolean
}) {
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  const time = tx.transaction_date.split('T')[1]?.slice(0, 5) || ''
  
  const [pressTimer, setPressTimer] = useState<ReturnType<typeof setTimeout> | null>(null)
  
  const handleTouchStart = () => {
    const timer = setTimeout(() => {
      onLongPress()
    }, 500)
    setPressTimer(timer)
  }
  
  const handleTouchEnd = () => {
    if (pressTimer) {
      clearTimeout(pressTimer)
      setPressTimer(null)
    }
  }
  
  return (
    <div 
      className={`${styles.item} ${selected ? styles.selected : ''}`}
      onClick={onClick}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      onTouchCancel={handleTouchEnd}
    >
      {selectMode && (
        <div className={`${styles.checkbox} ${selected ? styles.checked : ''}`}>
          {selected && <Check size={14} />}
        </div>
      )}
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
  )
}
