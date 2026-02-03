import { Moon, Sun, CreditCard, Tag, Smartphone, Palette } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTheme } from '../contexts/ThemeContext'
import Header from '../components/Header'
import styles from './Settings.module.css'

const colorPresets = [
  '#ff9500', '#ff6b35', '#ff3b30', '#ff2d55', '#af52de', 
  '#5856d6', '#007aff', '#0a84ff', '#5ac8fa', '#34c759',
  '#30d158', '#32ade6', '#ff9f0a', '#ffd60a'
]

export default function Settings() {
  const navigate = useNavigate()
  const { theme, toggleTheme, billingColors, setBillingColor } = useTheme()
  
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
        <h3 className={styles.sectionTitle}>
          <Palette size={16} />
          <span>결제예정 배경색 (라이트모드)</span>
        </h3>
        <div className={styles.colorPicker}>
          {colorPresets.map(color => (
            <button
              key={`light-${color}`}
              className={`${styles.colorBtn} ${billingColors.light === color ? styles.selected : ''}`}
              style={{ background: color }}
              onClick={() => setBillingColor('light', color)}
            />
          ))}
          <label className={styles.customColor}>
            <input
              type="color"
              value={billingColors.light}
              onChange={(e) => setBillingColor('light', e.target.value)}
            />
            <span>+</span>
          </label>
        </div>
      </div>
      
      <div className={styles.section}>
        <h3 className={styles.sectionTitle}>
          <Palette size={16} />
          <span>결제예정 배경색 (다크모드)</span>
        </h3>
        <div className={styles.colorPicker}>
          {colorPresets.map(color => (
            <button
              key={`dark-${color}`}
              className={`${styles.colorBtn} ${billingColors.dark === color ? styles.selected : ''}`}
              style={{ background: color }}
              onClick={() => setBillingColor('dark', color)}
            />
          ))}
          <label className={styles.customColor}>
            <input
              type="color"
              value={billingColors.dark}
              onChange={(e) => setBillingColor('dark', e.target.value)}
            />
            <span>+</span>
          </label>
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
        <div className={styles.item} onClick={() => navigate('/shortcut')}>
          <div className={styles.itemLeft}>
            <Smartphone size={20} />
            <span>아이폰 단축어 연동</span>
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
