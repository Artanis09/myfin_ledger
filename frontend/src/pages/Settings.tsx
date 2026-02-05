import { useState, useEffect } from 'react'
import { Moon, Sun, CreditCard, Tag, Smartphone, Palette, ChevronRight, MessageSquare, Calendar } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTheme } from '../contexts/ThemeContext'
import Header from '../components/Header'
import { api } from '../api'
import styles from './Settings.module.css'

const colorPresets = [
  '#ff9500', '#ff6b35', '#ff3b30', '#ff2d55', '#af52de', 
  '#5856d6', '#007aff', '#0a84ff', '#5ac8fa', '#34c759',
  '#30d158', '#32ade6', '#ff9f0a', '#ffd60a'
]

export default function Settings() {
  const navigate = useNavigate()
  const { theme, toggleTheme, billingColors, setBillingColor, showMemo, setShowMemo } = useTheme()
  const [showColorPicker, setShowColorPicker] = useState<'light' | 'dark' | null>(null)
  const [showPeriodSetting, setShowPeriodSetting] = useState(false)
  const [ledgerStartDay, setLedgerStartDay] = useState('1')
  const [ledgerEndDay, setLedgerEndDay] = useState('31')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.v2.getSettings().then((data) => {
      setLedgerStartDay(data.ledger_period_start_day || '1')
      setLedgerEndDay(data.ledger_period_end_day || '31')
    }).catch(console.error)
  }, [])

  const handleSavePeriod = async () => {
    setSaving(true)
    try {
      await api.v2.updateSettings({
        ledger_period_start_day: ledgerStartDay,
        ledger_period_end_day: ledgerEndDay,
      })
      setShowPeriodSetting(false)
    } catch (err) {
      alert('저장에 실패했습니다.')
    } finally {
      setSaving(false)
    }
  }

  const days = Array.from({ length: 31 }, (_, i) => i + 1)
  
  return (
    <div className={styles.page}>
      <Header title="설정" />
      
      <div className={styles.section}>
        <h3 className={styles.sectionTitle}>화면</h3>
        <div className={styles.item} onClick={toggleTheme}>
          <div className={styles.itemLeft}>
            {theme === 'dark' ? <Moon size={20} /> : <Sun size={20} />}
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>다크모드</span>
              <span className={styles.itemSubtitle}>{theme === 'dark' ? '활성화됨' : '비활성화'}</span>
            </div>
          </div>
          <div className={`${styles.toggle} ${theme === 'dark' ? styles.active : ''}`}>
            <div className={styles.toggleThumb} />
          </div>
        </div>
        
        <div className={styles.item} onClick={() => setShowMemo(!showMemo)}>
          <div className={styles.itemLeft}>
            <MessageSquare size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>메모 표시</span>
              <span className={styles.itemSubtitle}>거래 내역에 메모 표시</span>
            </div>
          </div>
          <div className={`${styles.toggle} ${showMemo ? styles.active : ''}`}>
            <div className={styles.toggleThumb} />
          </div>
        </div>
        
        <div className={styles.item} onClick={() => setShowColorPicker(showColorPicker === 'light' ? null : 'light')}>
          <div className={styles.itemLeft}>
            <Palette size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>결제예정 배경색 (라이트모드)</span>
              <span className={styles.itemSubtitle}>색상 선택</span>
            </div>
          </div>
          <div className={styles.itemRight}>
            <div className={styles.colorPreview} style={{ background: billingColors.light }} />
            <ChevronRight size={20} className={`${styles.chevron} ${showColorPicker === 'light' ? styles.open : ''}`} />
          </div>
        </div>
        {showColorPicker === 'light' && (
          <div className={styles.colorPickerWrapper}>
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
        )}
        
        <div className={styles.item} onClick={() => setShowColorPicker(showColorPicker === 'dark' ? null : 'dark')}>
          <div className={styles.itemLeft}>
            <Palette size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>결제예정 배경색 (다크모드)</span>
              <span className={styles.itemSubtitle}>색상 선택</span>
            </div>
          </div>
          <div className={styles.itemRight}>
            <div className={styles.colorPreview} style={{ background: billingColors.dark }} />
            <ChevronRight size={20} className={`${styles.chevron} ${showColorPicker === 'dark' ? styles.open : ''}`} />
          </div>
        </div>
        {showColorPicker === 'dark' && (
          <div className={styles.colorPickerWrapper}>
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
        )}
      </div>
      
      <div className={styles.section}>
        <h3 className={styles.sectionTitle}>가계부 설정</h3>
        <div className={styles.item} onClick={() => setShowPeriodSetting(!showPeriodSetting)}>
          <div className={styles.itemLeft}>
            <Calendar size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>관리 기간</span>
              <span className={styles.itemSubtitle}>매월 {ledgerStartDay}일 ~ {ledgerEndDay}일</span>
            </div>
          </div>
          <ChevronRight size={20} className={`${styles.chevron} ${showPeriodSetting ? styles.open : ''}`} />
        </div>
        {showPeriodSetting && (
          <div className={styles.periodSettingWrapper}>
            <div className={styles.periodSetting}>
              <div className={styles.periodRow}>
                <label>시작일</label>
                <select value={ledgerStartDay} onChange={e => setLedgerStartDay(e.target.value)}>
                  {days.map(d => <option key={d} value={d}>{d}일</option>)}
                </select>
              </div>
              <div className={styles.periodRow}>
                <label>종료일</label>
                <select value={ledgerEndDay} onChange={e => setLedgerEndDay(e.target.value)}>
                  {days.map(d => <option key={d} value={d}>{d}일</option>)}
                </select>
              </div>
              <button className={styles.saveBtn} onClick={handleSavePeriod} disabled={saving}>
                {saving ? '저장 중...' : '저장'}
              </button>
              <p className={styles.periodNote}>
                ※ 관리 기간이 월을 넘어갈 수 있습니다 (예: 23일 ~ 22일)
              </p>
            </div>
          </div>
        )}
      </div>
      
      <div className={styles.section}>
        <h3 className={styles.sectionTitle}>관리</h3>
        <div className={styles.item} onClick={() => navigate('/cards')}>
          <div className={styles.itemLeft}>
            <CreditCard size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>카드 관리</span>
              <span className={styles.itemSubtitle}>카드 추가, 수정, 삭제</span>
            </div>
          </div>
          <ChevronRight size={20} className={styles.arrow} />
        </div>
        <div className={styles.item} onClick={() => navigate('/categories')}>
          <div className={styles.itemLeft}>
            <Tag size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>카테고리 관리</span>
              <span className={styles.itemSubtitle}>지출 카테고리 설정</span>
            </div>
          </div>
          <ChevronRight size={20} className={styles.arrow} />
        </div>
        <div className={styles.item} onClick={() => navigate('/shortcut')}>
          <div className={styles.itemLeft}>
            <Smartphone size={20} />
            <div className={styles.itemText}>
              <span className={styles.itemTitle}>아이폰 단축어 연동</span>
              <span className={styles.itemSubtitle}>자동화 설정</span>
            </div>
          </div>
          <ChevronRight size={20} className={styles.arrow} />
        </div>
      </div>
      
      <div className={styles.version}>
        가계부 v1.0.0
      </div>
    </div>
  )
}
