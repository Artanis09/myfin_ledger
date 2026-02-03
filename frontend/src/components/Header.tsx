import { ChevronLeft, ChevronRight } from 'lucide-react'
import styles from './Header.module.css'

interface HeaderProps {
  title: string
  month?: Date
  onPrevMonth?: () => void
  onNextMonth?: () => void
  rightAction?: React.ReactNode
}

export default function Header({ title, month, onPrevMonth, onNextMonth, rightAction }: HeaderProps) {
  const formatMonth = (date: Date) => {
    return `${date.getFullYear()}년 ${date.getMonth() + 1}월`
  }
  
  return (
    <header className={styles.header}>
      <div className={styles.safeArea} />
      <div className={styles.titleRow}>
        <h1 className={styles.pageTitle}>{title}</h1>
        {rightAction && <div className={styles.rightAction}>{rightAction}</div>}
      </div>
      {month && (
        <div className={styles.monthRow}>
          <button className={styles.navBtn} onClick={onPrevMonth} aria-label="이전 달">
            <ChevronLeft size={20} />
          </button>
          <span className={styles.monthText}>{formatMonth(month)}</span>
          <button className={styles.navBtn} onClick={onNextMonth} aria-label="다음 달">
            <ChevronRight size={20} />
          </button>
        </div>
      )}
    </header>
  )
}
