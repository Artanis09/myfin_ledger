import { useState, useEffect } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import Header from '../components/Header'
import { api } from '../api'
import styles from './SMSRules.module.css'

interface SMSRule {
  id: number
  rule_type: string
  keyword: string
  tx_type: string | null
  priority: number
  description: string | null
}

export default function SMSRules() {
  const [rules, setRules] = useState<SMSRule[]>([])
  const [showForm, setShowForm] = useState(false)
  const [newRule, setNewRule] = useState({
    rule_type: 'card',
    keyword: '',
    tx_type: 'expense',
    priority: 50,
    description: ''
  })

  useEffect(() => {
    loadRules()
  }, [])

  const loadRules = async () => {
    try {
      const data = await api.v2.getSMSMappingRules()
      setRules(data.rules || [])
    } catch (err) {
      console.error('Failed to load rules:', err)
    }
  }

  const handleCreate = async () => {
    if (!newRule.keyword.trim()) {
      alert('키워드를 입력해주세요')
      return
    }
    try {
      await api.v2.createSMSMappingRule({
        rule_type: newRule.rule_type,
        keyword: newRule.keyword,
        tx_type: newRule.tx_type,
        priority: newRule.priority,
        description: newRule.description || undefined
      })
      setNewRule({ rule_type: 'card', keyword: '', tx_type: 'expense', priority: 50, description: '' })
      setShowForm(false)
      loadRules()
    } catch (err) {
      alert('추가에 실패했습니다')
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('정말 삭제하시겠습니까?')) return
    try {
      await api.v2.deleteSMSMappingRule(id)
      loadRules()
    } catch (err) {
      alert('삭제에 실패했습니다')
    }
  }

  const getRuleTypeLabel = (type: string) => {
    switch (type) {
      case 'card': return '카드'
      case 'income': return '입금'
      case 'expense': return '출금'
      default: return type
    }
  }

  const getTxTypeLabel = (type: string | null) => {
    if (!type) return '-'
    return type === 'income' ? '수입' : '지출'
  }

  return (
    <div className={styles.page}>
      <Header 
        title="문자 파싱 키워드"
        rightAction={
          <button className={styles.addBtn} onClick={() => setShowForm(true)}>
            <Plus size={24} />
          </button>
        }
      />
      
      <div className={styles.description}>
        문자 메시지에 포함된 키워드로 거래 유형을 판별합니다.
      </div>
      
      <div className={styles.list}>
        {rules.map(rule => (
          <div key={rule.id} className={styles.item}>
            <div className={styles.itemMain}>
              <div className={styles.keyword}>"{rule.keyword}"</div>
              <div className={styles.itemMeta}>
                <span className={`${styles.badge} ${styles[rule.rule_type]}`}>
                  {getRuleTypeLabel(rule.rule_type)}
                </span>
                <span className={styles.txType}>→ {getTxTypeLabel(rule.tx_type)}</span>
                <span className={styles.priority}>우선순위: {rule.priority}</span>
              </div>
              {rule.description && (
                <div className={styles.desc}>{rule.description}</div>
              )}
            </div>
            <button className={styles.deleteBtn} onClick={() => handleDelete(rule.id)}>
              <Trash2 size={18} />
            </button>
          </div>
        ))}
        {rules.length === 0 && (
          <div className={styles.empty}>등록된 키워드가 없습니다</div>
        )}
      </div>
      
      {showForm && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <h3>키워드 추가</h3>
            
            <div className={styles.formGroup}>
              <label>키워드</label>
              <input
                type="text"
                placeholder="예: 승인, 입금, 출금"
                value={newRule.keyword}
                onChange={e => setNewRule({ ...newRule, keyword: e.target.value })}
              />
            </div>
            
            <div className={styles.formGroup}>
              <label>판별 유형</label>
              <select
                value={newRule.rule_type}
                onChange={e => setNewRule({ ...newRule, rule_type: e.target.value })}
              >
                <option value="card">카드 내역</option>
                <option value="income">입금 (수입)</option>
                <option value="expense">출금 (지출)</option>
              </select>
            </div>
            
            <div className={styles.formGroup}>
              <label>거래 유형</label>
              <select
                value={newRule.tx_type}
                onChange={e => setNewRule({ ...newRule, tx_type: e.target.value })}
              >
                <option value="expense">지출</option>
                <option value="income">수입</option>
              </select>
            </div>
            
            <div className={styles.formGroup}>
              <label>우선순위 (1-100)</label>
              <input
                type="number"
                min="1"
                max="100"
                value={newRule.priority}
                onChange={e => setNewRule({ ...newRule, priority: parseInt(e.target.value) || 50 })}
              />
            </div>
            
            <div className={styles.formGroup}>
              <label>설명 (선택)</label>
              <input
                type="text"
                placeholder="예: 카드 승인 키워드"
                value={newRule.description}
                onChange={e => setNewRule({ ...newRule, description: e.target.value })}
              />
            </div>
            
            <div className={styles.modalActions}>
              <button className={styles.cancelBtn} onClick={() => setShowForm(false)}>취소</button>
              <button className={styles.confirmBtn} onClick={handleCreate}>추가</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
