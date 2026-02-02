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
      {month ? (
        <>
          <button className={styles.navBtn} onClick={onPrevMonth}>
            <ChevronLeft size={24} />
          </button>
          <h1 className={styles.title}>{formatMonth(month)}</h1>
          <button className={styles.navBtn} onClick={onNextMonth}>
            <ChevronRight size={24} />
          </button>
        </>
      ) : (
        <>
          <h1 className={styles.title}>{title}</h1>
          {rightAction && <div className={styles.rightAction}>{rightAction}</div>}
        </>
      )}
    </header>
  )
}
