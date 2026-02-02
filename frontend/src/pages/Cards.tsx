import { useState, useEffect } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import Header from '../components/Header'
import { api } from '../api'
import type { Card } from '../types'
import styles from './Cards.module.css'

export default function Cards() {
  const [cards, setCards] = useState<Card[]>([])
  const [showForm, setShowForm] = useState(false)
  const [newCard, setNewCard] = useState({ name: '', billing_start_day: '1', billing_end_day: '31' })
  
  useEffect(() => {
    loadCards()
  }, [])
  
  const loadCards = async () => {
    try {
      const data = await api.getCards()
      setCards(data.cards || [])
    } catch (err) {
      console.error('Failed to load:', err)
    }
  }
  
  const handleCreate = async () => {
    if (!newCard.name.trim()) return
    try {
      await api.createCard({
        name: newCard.name,
        billing_start_day: Number(newCard.billing_start_day),
        billing_end_day: Number(newCard.billing_end_day),
      })
      setNewCard({ name: '', billing_start_day: '1', billing_end_day: '31' })
      setShowForm(false)
      loadCards()
    } catch (err) {
      console.error('Failed to create:', err)
    }
  }
  
  const handleUpdate = async (card: Card) => {
    try {
      await api.updateCard(card.id, card)
    } catch (err) {
      console.error('Failed to update:', err)
    }
  }
  
  const handleDelete = async (id: number) => {
    if (!confirm('정말 삭제하시겠습니까?')) return
    try {
      await api.deleteCard(id)
      loadCards()
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }
  
  return (
    <div className={styles.page}>
      <Header 
        title="카드 관리"
        rightAction={
          <button className={styles.addBtn} onClick={() => setShowForm(true)}>
            <Plus size={24} />
          </button>
        }
      />
      
      <div className={styles.description}>
        각 카드의 결산일 기간을 설정하면 월별 예상 결제 금액을 계산합니다.
      </div>
      
      <div className={styles.list}>
        {cards.map(card => (
          <div key={card.id} className={styles.item}>
            <div className={styles.itemMain}>
              <input
                className={styles.cardName}
                value={card.name}
                onChange={e => {
                  const updated = { ...card, name: e.target.value }
                  setCards(prev => prev.map(c => c.id === card.id ? updated : c))
                }}
                onBlur={() => handleUpdate(card)}
              />
              <div className={styles.billingPeriod}>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={card.billing_start_day}
                  onChange={e => {
                    const updated = { ...card, billing_start_day: Number(e.target.value) }
                    setCards(prev => prev.map(c => c.id === card.id ? updated : c))
                  }}
                  onBlur={() => handleUpdate(card)}
                />
                <span>일 ~</span>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={card.billing_end_day}
                  onChange={e => {
                    const updated = { ...card, billing_end_day: Number(e.target.value) }
                    setCards(prev => prev.map(c => c.id === card.id ? updated : c))
                  }}
                  onBlur={() => handleUpdate(card)}
                />
                <span>일</span>
              </div>
            </div>
            <button className={styles.deleteBtn} onClick={() => handleDelete(card.id)}>
              <Trash2 size={18} />
            </button>
          </div>
        ))}
      </div>
      
      {showForm && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <h3>새 카드 추가</h3>
            <input
              type="text"
              placeholder="카드사 이름"
              value={newCard.name}
              onChange={e => setNewCard(prev => ({ ...prev, name: e.target.value }))}
            />
            <div className={styles.billingInputs}>
              <div>
                <label>결산 시작일</label>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={newCard.billing_start_day}
                  onChange={e => setNewCard(prev => ({ ...prev, billing_start_day: e.target.value }))}
                />
              </div>
              <div>
                <label>결산 종료일</label>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={newCard.billing_end_day}
                  onChange={e => setNewCard(prev => ({ ...prev, billing_end_day: e.target.value }))}
                />
              </div>
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
