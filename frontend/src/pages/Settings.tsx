import { Moon, Sun, CreditCard, Tag } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTheme } from '../contexts/ThemeContext'
import Header from '../components/Header'
import styles from './Settings.module.css'

export default function Settings() {
  const navigate = useNavigate()
  const { theme, toggleTheme } = useTheme()
  
  return (
    <div className={styles.page}>
      <Header title="설정" />
      
      <div className={styles.section}>
        <h3 className={styles.sectionTitle}>화면</h3>
        <div className={styles.item} onClick={toggleTheme}>
          <div className={styles.itemLeft}>
            {theme === 'dark' ? <Moon size={20} /> : <Sun size={20} />}
            <span>다크모드</span>
          </div>
          <div className={`${styles.toggle} ${theme === 'dark' ? styles.active : ''}`}>
            <div className={styles.toggleThumb} />
          </div>
        </div>
      </div>
      
      <div className={styles.section}>
        <h3 className={styles.sectionTitle}>관리</h3>
        <div className={styles.item} onClick={() => navigate('/cards')}>
          <div className={styles.itemLeft}>
            <CreditCard size={20} />
            <span>카드 관리</span>
          </div>
          <span className={styles.arrow}>›</span>
        </div>
        <div className={styles.item} onClick={() => navigate('/categories')}>
          <div className={styles.itemLeft}>
            <Tag size={20} />
            <span>카테고리 관리</span>
          </div>
          <span className={styles.arrow}>›</span>
        </div>
      </div>
      
      <div className={styles.version}>
        가계부 v1.0.0
      </div>
    </div>
  )
}
