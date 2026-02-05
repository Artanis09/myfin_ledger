import { ChevronLeft, ChevronRight, Info } from 'lucide-react'
import styles from './Header.module.css'

interface HeaderProps {
  title: string
  month?: Date
  onPrevMonth?: () => void
  onNextMonth?: () => void
  rightAction?: React.ReactNode
  onInfoClick?: () => void
}

export default function Header({ title, month, onPrevMonth, onNextMonth, rightAction, onInfoClick }: HeaderProps) {
  const formatMonth = (date: Date) => {
    return `${date.getFullYear()}년 ${date.getMonth() + 1}월`
  }
  
  return (
    <header className={styles.header}>
      <div className={styles.safeArea} />
      <div className={styles.titleRow}>
        <div className={styles.titleWithInfo}>
          <h1 className={styles.pageTitle}>{title}</h1>
          {onInfoClick && (
            <button className={styles.infoBtn} onClick={onInfoClick} aria-label="정보">
              <Info size={18} />
            </button>
          )}
        </div>
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
