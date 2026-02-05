import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { format } from 'date-fns'
import { ChevronLeft, Minus, Plus, Trash2 } from 'lucide-react'
import { api } from '../api'
import type { AssetType, Category, IncomeCategory } from '../types'
import styles from './TransactionFormV2.module.css'

type TxType = 'expense' | 'income'

export default function TransactionFormV2() {
  const navigate = useNavigate()
  
  const { id } = useParams()
  const isEdit = !!id

  const [txType, setTxType] = useState<TxType>('expense')
  const [assetTypeId, setAssetTypeId] = useState<number | ''>('')
  const [date, setDate] = useState(format(new Date(), 'yyyy-MM-dd'))
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [categoryId, setCategoryId] = useState<number | ''>('')
  const [incomeCategoryId, setIncomeCategoryId] = useState<number | ''>('')
  const [memo, setMemo] = useState('')
  const [isRecurring, setIsRecurring] = useState(false)
  
  const [expenseAssets, setExpenseAssets] = useState<AssetType[]>([])
  const [incomeAssets, setIncomeAssets] = useState<AssetType[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [incomeCategories, setIncomeCategories] = useState<IncomeCategory[]>([])
  
  const [showAddAsset, setShowAddAsset] = useState(false)
  const [showAddCategory, setShowAddCategory] = useState(false)
  const [newItemName, setNewItemName] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [loading, setLoading] = useState(isEdit)

  useEffect(() => {
    loadData()
  }, [])

  useEffect(() => {
    if (isEdit && id) {
      loadTransaction(Number(id))
    }
  }, [isEdit, id])

  const loadData = async () => {
    try {
      const [expRes, incRes, catRes, incCatRes] = await Promise.all([
        api.v2.getAssetTypes('expense'),
        api.v2.getAssetTypes('income'),
        api.getCategories(),
        api.v2.getIncomeCategories()
      ])
      // 고정지출/고정수입 자산 유형 필터링 (is_recurring = 1인 것 제외)
      const expenseFiltered = (expRes.asset_types || []).filter((a: AssetType) => !a.is_recurring)
      const incomeFiltered = (incRes.asset_types || []).filter((a: AssetType) => !a.is_recurring)
      setExpenseAssets(expenseFiltered)
      setIncomeAssets(incomeFiltered)
      setCategories(catRes.categories || [])
      setIncomeCategories(incCatRes.income_categories || [])
    } catch (err) {
      console.error('Failed to load data:', err)
    }
  }

  const loadTransaction = async (txId: number) => {
    try {
      const tx = await api.v2.getTransaction(txId)
      setTxType(tx.tx_type || 'expense')
      setAssetTypeId(tx.asset_type_id || '')
      setDate(tx.transaction_date?.substring(0, 10) || format(new Date(), 'yyyy-MM-dd'))
      setAmount(tx.amount ? new Intl.NumberFormat('ko-KR').format(tx.amount) : '')
      setDescription(tx.description || '')
      setCategoryId(tx.category_id || '')
      setIncomeCategoryId(tx.income_category_id || '')
      setMemo(tx.memo || '')
      setIsRecurring(tx.is_recurring === 1)
    } catch (err) {
      console.error('Failed to load transaction:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async () => {
    if (!assetTypeId || !amount || !description) {
      alert('필수 항목을 입력해주세요')
      return
    }
    setSubmitting(true)
    try {
      const data = {
        tx_type: txType,
        asset_type_id: assetTypeId,
        transaction_date: date,
        amount: parseInt(amount.replace(/,/g, '')),
        description,
        category_id: txType === 'expense' && categoryId ? categoryId : null,
        income_category_id: txType === 'income' && incomeCategoryId ? incomeCategoryId : null,
        memo: memo || null,
        is_installment: 0,
        is_cancelled: 0,
        is_recurring: isRecurring ? 1 : 0,
      }

      if (isEdit && id) {
        await api.v2.updateTransaction(Number(id), data)
      } else {
        await api.v2.createTransaction(data)
      }
      // 수정 후 이전 페이지로 돌아가기 (현재 페이지에 머무르기 위해 history back 사용)
      navigate(-1)
    } catch (err) {
      console.error('Failed to save:', err)
      alert('저장에 실패했습니다')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!confirm('정말 삭제하시겠습니까?')) return
    try {
      await api.v2.deleteTransaction(Number(id))
      navigate(-1)
    } catch (err) {
      console.error('Failed to delete:', err)
      alert('삭제에 실패했습니다')
    }
  }

  const handleAddAsset = async () => {
    if (!newItemName.trim()) return
    try {
      await api.v2.createAssetType({
        name: newItemName,
        type: txType,
        is_recurring: 0,
        display_order: 999
      })
      loadData()
      setShowAddAsset(false)
      setNewItemName('')
    } catch (err) {
      alert('추가에 실패했습니다')
    }
  }

  const handleAddCategory = async () => {
    if (!newItemName.trim()) return
    try {
      if (txType === 'expense') {
        await api.createCategory({ name: newItemName, keywords: '' })
      } else {
        await api.v2.createIncomeCategory({ name: newItemName, display_order: 99 })
      }
      loadData()
      setShowAddCategory(false)
      setNewItemName('')
    } catch (err) {
      alert('추가에 실패했습니다')
    }
  }

  const formatAmountInput = (value: string) => {
    const num = value.replace(/[^0-9]/g, '')
    if (!num) return ''
    return new Intl.NumberFormat('ko-KR').format(parseInt(num))
  }

  const currentAssets = txType === 'expense' ? expenseAssets : incomeAssets

  if (loading) {
    return (
      <div className={styles.page}>
        <div className={styles.header}>
          <button className={styles.backBtn} onClick={() => navigate(-1)}>
            <ChevronLeft size={24} />
          </button>
          <span className={styles.headerTitle}>로딩 중...</span>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <button className={styles.backBtn} onClick={() => navigate(-1)}>
          <ChevronLeft size={24} />
        </button>
        <span className={styles.headerTitle}>{isEdit ? '내역 수정' : '내역 등록'}</span>
        {isEdit && (
          <button className={styles.deleteBtn} onClick={handleDelete}>
            <Trash2 size={20} />
          </button>
        )}
      </div>

      <div className={styles.typeTabs}>
        <button 
          className={`${styles.typeTab} ${styles.expense} ${txType === 'expense' ? styles.active : ''}`}
          onClick={() => { setTxType('expense'); setAssetTypeId(''); setCategoryId('') }}
        >
          <Minus size={18} /> 지출
        </button>
        <button 
          className={`${styles.typeTab} ${styles.income} ${txType === 'income' ? styles.active : ''}`}
          onClick={() => { setTxType('income'); setAssetTypeId(''); setIncomeCategoryId('') }}
        >
          <Plus size={18} /> 수입
        </button>
      </div>

      <div className={styles.form}>
        <div className={styles.field}>
          <span className={styles.label}>자산</span>
          <div className={styles.assetRow}>
            <div className={styles.selectWrapper}>
              <select 
                className={styles.select}
                value={assetTypeId}
                onChange={(e) => {
                  if (e.target.value === 'add') {
                    setShowAddAsset(true)
                  } else {
                    setAssetTypeId(e.target.value ? Number(e.target.value) : '')
                  }
                }}
              >
                <option value="">선택해주세요</option>
                {currentAssets.map(a => (
                  <option key={a.id} value={a.id}>{a.name}</option>
                ))}
                <option value="add" className={styles.addOption}>+ 자산 추가</option>
              </select>
            </div>
            <label className={styles.recurringCheck}>
              <input 
                type="checkbox" 
                checked={isRecurring} 
                onChange={(e) => setIsRecurring(e.target.checked)} 
              />
              <span>고정{txType === 'expense' ? '지출' : '수입'}</span>
            </label>
          </div>
        </div>

        <div className={styles.field}>
          <span className={styles.label}>날짜</span>
          <input 
            type="date"
            className={styles.dateInput}
            value={date}
            onChange={(e) => setDate(e.target.value)}
          />
        </div>

        <div className={styles.field}>
          <span className={styles.label}>금액</span>
          <div className={styles.amountWrapper}>
            <input 
              type="text"
              inputMode="numeric"
              className={`${styles.amountInput} ${styles[txType]}`}
              placeholder="0"
              value={amount}
              onChange={(e) => setAmount(formatAmountInput(e.target.value))}
            />
            <span className={styles.amountUnit}>원</span>
          </div>
        </div>

        <div className={styles.field}>
          <span className={styles.label}>사용내역</span>
          <input 
            type="text"
            className={styles.input}
            placeholder="사용내역을 입력하세요"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        <div className={styles.field}>
          <span className={styles.label}>카테고리</span>
          <div className={styles.selectWrapper}>
            {txType === 'expense' ? (
              <select 
                className={styles.select}
                value={categoryId}
                onChange={(e) => {
                  if (e.target.value === 'add') {
                    setShowAddCategory(true)
                  } else {
                    setCategoryId(e.target.value ? Number(e.target.value) : '')
                  }
                }}
              >
                <option value="">선택해주세요</option>
                {categories.map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
                <option value="add">+ 카테고리 추가</option>
              </select>
            ) : (
              <select 
                className={styles.select}
                value={incomeCategoryId}
                onChange={(e) => {
                  if (e.target.value === 'add') {
                    setShowAddCategory(true)
                  } else {
                    setIncomeCategoryId(e.target.value ? Number(e.target.value) : '')
                  }
                }}
              >
                <option value="">선택해주세요</option>
                {incomeCategories.map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
                <option value="add">+ 카테고리 추가</option>
              </select>
            )}
          </div>
        </div>

        <div className={styles.field}>
          <span className={styles.label}>메모</span>
          <textarea 
            className={styles.textarea}
            placeholder="메모 (선택)"
            value={memo}
            onChange={(e) => setMemo(e.target.value)}
          />
        </div>
      </div>

      <button 
        className={`${styles.submitBtn} ${styles[txType]}`}
        onClick={handleSubmit}
        disabled={submitting || !assetTypeId || !amount || !description}
      >
        {submitting ? '저장 중...' : (isEdit ? '저장' : (txType === 'expense' ? '지출 등록' : '수입 등록'))}
      </button>

      {showAddAsset && (
        <>
          <div className={styles.overlay} onClick={() => setShowAddAsset(false)} />
          <div className={styles.modal}>
            <h3>자산 추가</h3>
            <input 
              type="text"
              placeholder="자산명"
              value={newItemName}
              onChange={(e) => setNewItemName(e.target.value)}
            />
            <div className={styles.modalBtns}>
              <button className={styles.cancel} onClick={() => setShowAddAsset(false)}>취소</button>
              <button className={styles.confirm} onClick={handleAddAsset}>추가</button>
            </div>
          </div>
        </>
      )}

      {showAddCategory && (
        <>
          <div className={styles.overlay} onClick={() => setShowAddCategory(false)} />
          <div className={styles.modal}>
            <h3>카테고리 추가</h3>
            <input 
              type="text"
              placeholder="카테고리명"
              value={newItemName}
              onChange={(e) => setNewItemName(e.target.value)}
            />
            <div className={styles.modalBtns}>
              <button className={styles.cancel} onClick={() => setShowAddCategory(false)}>취소</button>
              <button className={styles.confirm} onClick={handleAddCategory}>추가</button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
