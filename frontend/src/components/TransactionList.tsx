import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import styles from './TransactionList.module.css'
import type { Transaction } from '../types'

interface TransactionListProps {
  transactions: Transaction[]
  onEdit?: (id: number) => void
  onDelete?: (id: number) => void
  grouped?: boolean
}

export default function TransactionList({ transactions, onEdit, grouped = true }: TransactionListProps) {
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
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function TransactionItem({ tx, onEdit }: { 
  tx: Transaction
  onEdit?: (id: number) => void
}) {
  const formatMoney = (amount: number) => amount.toLocaleString() + '원'
  const time = tx.transaction_date.split('T')[1]?.slice(0, 5) || ''
  
  return (
    <div 
      className={styles.item}
      onClick={() => onEdit?.(tx.id)}
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
  )
}
