import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, MessageSquare } from 'lucide-react'
import { api } from '../api'
import type { Card, Category } from '../types'
import styles from './TransactionForm.module.css'

export default function TransactionForm() {
  const navigate = useNavigate()
  const { id } = useParams()
  const isEdit = !!id
  
  const [cards, setCards] = useState<Card[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [smsText, setSmsText] = useState('')
  const [showSmsInput, setShowSmsInput] = useState(false)
  
  const [form, setForm] = useState({
    card_id: '',
    transaction_date: new Date().toISOString().split('T')[0],
    amount: '',
    description: '',
    category_id: '',
    is_installment: false,
    installment_months: '',
    installment_current: '1',
    original_amount: '',
  })
  
  useEffect(() => {
    loadInitialData()
  }, [])
  
  const loadInitialData = async () => {
    try {
      const [cardsData, catsData] = await Promise.all([
        api.getCards(),
        api.getCategories(),
      ])
      setCards(cardsData.cards || [])
      setCategories(catsData.categories || [])
      
      if (cardsData.cards?.length > 0) {
        setForm(prev => ({ ...prev, card_id: String(cardsData.cards[0].id) }))
      }
      
      if (isEdit) {
        const tx = await api.getTransaction(Number(id))
        setForm({
          card_id: String(tx.card_id),
          transaction_date: tx.transaction_date.split('T')[0],
          amount: String(tx.amount),
          description: tx.description,
          category_id: tx.category_id ? String(tx.category_id) : '',
          is_installment: tx.is_installment === 1,
          installment_months: tx.installment_months ? String(tx.installment_months) : '',
          installment_current: tx.installment_current ? String(tx.installment_current) : '1',
          original_amount: tx.original_amount ? String(tx.original_amount) : '',
        })
      }
    } catch (err) {
      console.error('Failed to load:', err)
    }
  }
  
  const handleParseSMS = async () => {
    if (!smsText.trim()) return
    try {
      const result = await api.parseSMS(smsText)
      
      const card = cards.find(c => 
        c.name.includes(result.card_name) || result.card_name.includes(c.name)
      )
      
      setForm(prev => ({
        ...prev,
        card_id: card ? String(card.id) : prev.card_id,
        transaction_date: result.date || prev.transaction_date,
        amount: result.amount ? String(result.amount) : prev.amount,
        description: result.description || prev.description,
        is_installment: result.is_installment,
        installment_months: result.installment_months ? String(result.installment_months) : '',
      }))
      setShowSmsInput(false)
    } catch (err) {
      console.error('Failed to parse:', err)
    }
  }
  
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    const data = {
      card_id: Number(form.card_id),
      transaction_date: form.transaction_date,
      amount: Number(form.amount.replace(/,/g, '')),
      description: form.description,
      category_id: form.category_id ? Number(form.category_id) : null,
      is_installment: form.is_installment ? 1 : 0,
      installment_months: form.is_installment ? Number(form.installment_months) : null,
      installment_current: form.is_installment ? Number(form.installment_current) : null,
      original_amount: form.is_installment ? Number(form.original_amount.replace(/,/g, '')) : null,
    }
    
    try {
      if (isEdit) {
        await api.updateTransaction(Number(id), data)
      } else {
        await api.createTransaction(data)
      }
      navigate('/transactions')
    } catch (err) {
      console.error('Failed to save:', err)
    }
  }
  
  const handleDelete = async () => {
    if (!confirm('정말 삭제하시겠습니까?')) return
    try {
      await api.deleteTransaction(Number(id))
      navigate('/transactions')
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }
  
  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <button className={styles.backBtn} onClick={() => navigate(-1)}>
          <ArrowLeft size={24} />
        </button>
        <h1>{isEdit ? '거래 수정' : '새 거래'}</h1>
        {!isEdit && (
          <button 
            className={styles.smsBtn}
            onClick={() => setShowSmsInput(!showSmsInput)}
          >
            <MessageSquare size={20} />
          </button>
        )}
      </header>
      
      {showSmsInput && (
        <div className={styles.smsSection}>
          <textarea
            className={styles.smsInput}
            placeholder="결제 문자를 붙여넣으세요..."
            value={smsText}
            onChange={e => setSmsText(e.target.value)}
            rows={4}
          />
          <button className={styles.parseBtn} onClick={handleParseSMS}>
            문자 분석
          </button>
        </div>
      )}
      
      <form className={styles.form} onSubmit={handleSubmit}>
        <div className={styles.formGroup}>
          <label>카드사</label>
          <select
            value={form.card_id}
            onChange={e => setForm(prev => ({ ...prev, card_id: e.target.value }))}
            required
          >
            {cards.map(card => (
              <option key={card.id} value={card.id}>{card.name}</option>
            ))}
          </select>
        </div>
        
        <div className={styles.formGroup}>
          <label>날짜</label>
          <input
            type="date"
            value={form.transaction_date}
            onChange={e => setForm(prev => ({ ...prev, transaction_date: e.target.value }))}
            required
          />
        </div>
        
        <div className={styles.formGroup}>
          <label>금액</label>
          <input
            type="text"
            inputMode="numeric"
            placeholder="0"
            value={form.amount}
            onChange={e => setForm(prev => ({ ...prev, amount: e.target.value }))}
            required
          />
        </div>
        
        <div className={styles.formGroup}>
          <label>사용내역</label>
          <input
            type="text"
            placeholder="예) 쿠팡"
            value={form.description}
            onChange={e => setForm(prev => ({ ...prev, description: e.target.value }))}
            required
          />
        </div>
        
        <div className={styles.formGroup}>
          <label>카테고리</label>
          <select
            value={form.category_id}
            onChange={e => setForm(prev => ({ ...prev, category_id: e.target.value }))}
          >
            <option value="">자동 분류</option>
            {categories.map(cat => (
              <option key={cat.id} value={cat.id}>{cat.name}</option>
            ))}
          </select>
        </div>
        
        <div className={styles.formGroup}>
          <label className={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={form.is_installment}
              onChange={e => setForm(prev => ({ ...prev, is_installment: e.target.checked }))}
            />
            <span>할부 결제</span>
          </label>
        </div>
        
        {form.is_installment && (
          <div className={styles.installmentFields}>
            <div className={styles.formGroup}>
              <label>할부 개월</label>
              <input
                type="number"
                min="2"
                max="36"
                value={form.installment_months}
                onChange={e => setForm(prev => ({ ...prev, installment_months: e.target.value }))}
              />
            </div>
            <div className={styles.formGroup}>
              <label>현재 회차</label>
              <input
                type="number"
                min="1"
                value={form.installment_current}
                onChange={e => setForm(prev => ({ ...prev, installment_current: e.target.value }))}
              />
            </div>
            <div className={styles.formGroup}>
              <label>원금 총액</label>
              <input
                type="text"
                inputMode="numeric"
                value={form.original_amount}
                onChange={e => setForm(prev => ({ ...prev, original_amount: e.target.value }))}
              />
            </div>
          </div>
        )}
        
        <div className={styles.actions}>
          <button type="submit" className={styles.submitBtn}>
            {isEdit ? '저장' : '등록'}
          </button>
          {isEdit && (
            <button type="button" className={styles.deleteBtn} onClick={handleDelete}>
              삭제
            </button>
          )}
        </div>
      </form>
    </div>
  )
}
