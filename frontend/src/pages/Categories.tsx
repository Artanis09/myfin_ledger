import { useState, useEffect } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import Header from '../components/Header'
import { api } from '../api'
import type { Category } from '../types'
import styles from './Categories.module.css'

export default function Categories() {
  const [categories, setCategories] = useState<Category[]>([])
  const [showForm, setShowForm] = useState(false)
  const [newCategory, setNewCategory] = useState({ name: '', keywords: '' })
  
  useEffect(() => {
    loadCategories()
  }, [])
  
  const loadCategories = async () => {
    try {
      const data = await api.getCategories()
      setCategories(data.categories || [])
    } catch (err) {
      console.error('Failed to load:', err)
    }
  }
  
  const handleCreate = async () => {
    if (!newCategory.name.trim()) return
    try {
      await api.createCategory(newCategory)
      setNewCategory({ name: '', keywords: '' })
      setShowForm(false)
      loadCategories()
    } catch (err) {
      console.error('Failed to create:', err)
    }
  }
  
  const handleUpdate = async (cat: Category) => {
    try {
      await api.updateCategory(cat.id, cat)
    } catch (err) {
      console.error('Failed to update:', err)
    }
  }
  
  const handleDelete = async (id: number) => {
    if (!confirm('정말 삭제하시겠습니까?')) return
    try {
      await api.deleteCategory(id)
      loadCategories()
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }
  
  return (
    <div className={styles.page}>
      <Header 
        title="카테고리 관리"
        rightAction={
          <button className={styles.addBtn} onClick={() => setShowForm(true)}>
            <Plus size={24} />
          </button>
        }
      />
      
      <div className={styles.description}>
        키워드를 설정하면 거래내역이 자동으로 분류됩니다. (쉼표로 구분)
      </div>
      
      <div className={styles.list}>
        {categories.map(cat => (
          <div key={cat.id} className={styles.item}>
            <div className={styles.itemMain}>
              <input
                className={styles.catName}
                value={cat.name}
                onChange={e => {
                  const updated = { ...cat, name: e.target.value }
                  setCategories(prev => prev.map(c => c.id === cat.id ? updated : c))
                }}
                onBlur={() => handleUpdate(cat)}
              />
              <input
                className={styles.keywords}
                placeholder="키워드 입력 (쉼표 구분)"
                value={cat.keywords}
                onChange={e => {
                  const updated = { ...cat, keywords: e.target.value }
                  setCategories(prev => prev.map(c => c.id === cat.id ? updated : c))
                }}
                onBlur={() => handleUpdate(cat)}
              />
            </div>
            <button className={styles.deleteBtn} onClick={() => handleDelete(cat.id)}>
              <Trash2 size={18} />
            </button>
          </div>
        ))}
      </div>
      
      {showForm && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <h3>새 카테고리 추가</h3>
            <input
              type="text"
              placeholder="카테고리 이름"
              value={newCategory.name}
              onChange={e => setNewCategory(prev => ({ ...prev, name: e.target.value }))}
            />
            <input
              type="text"
              placeholder="키워드 (쉼표로 구분)"
              value={newCategory.keywords}
              onChange={e => setNewCategory(prev => ({ ...prev, keywords: e.target.value }))}
            />
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
