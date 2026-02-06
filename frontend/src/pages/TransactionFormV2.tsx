import { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { format } from 'date-fns'
import { ChevronLeft, Minus, Plus, Trash2, Sparkles, X, ClipboardPaste } from 'lucide-react'
import { api } from '../api'
import type { AssetType, Category, IncomeCategory } from '../types'
import styles from './TransactionFormV2.module.css'

type TxType = 'expense' | 'income'

// 수입 자산 기본값
const INCOME_ASSETS: AssetType[] = [
  { id: -1, name: '입금', type: 'income', is_recurring: 0, display_order: 1, is_card: 0, card_id: null, is_system: 1 },
  { id: -2, name: '현금', type: 'income', is_recurring: 0, display_order: 2, is_card: 0, card_id: null, is_system: 1 },
]

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
  const [isAutoRepeat, setIsAutoRepeat] = useState(false)
  const [showRepeatTooltip, setShowRepeatTooltip] = useState(false)
  
  const [expenseAssets, setExpenseAssets] = useState<AssetType[]>([])
  const [incomeAssets] = useState<AssetType[]>(INCOME_ASSETS)
  const [categories, setCategories] = useState<Category[]>([])
  const [incomeCategories, setIncomeCategories] = useState<IncomeCategory[]>([])
  
  // Category mapping tooltip
  const [mappingTooltip, setMappingTooltip] = useState<{ show: boolean; categoryName: string } | null>(null)
  const [originalCategoryId, setOriginalCategoryId] = useState<number | ''>('')
  const tooltipTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  
  // Modal states
  const [showAssetModal, setShowAssetModal] = useState(false)
  const [showCategoryModal, setShowCategoryModal] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [loading, setLoading] = useState(isEdit)
  
  // SMS 붙여넣기
  const [smsText, setSmsText] = useState('')
  const [showSmsModal, setShowSmsModal] = useState(false)

  useEffect(() => {
    // 페이지 진입 시 스크롤을 맨 위로
    window.scrollTo(0, 0)
    loadData()
  }, [])

  useEffect(() => {
    if (isEdit && id) {
      loadTransaction(Number(id))
    }
  }, [isEdit, id])

  const loadData = async () => {
    try {
      const [expRes, catRes, incCatRes] = await Promise.all([
        api.v2.getAssetTypes('expense'),
        api.getCategories(),
        api.v2.getIncomeCategories()
      ])
      const expenseFiltered = (expRes.asset_types || []).filter((a: AssetType) => !a.is_recurring)
      setExpenseAssets(expenseFiltered)
      
      // 카테고리에 '미분류' 추가 (없으면)
      const cats = catRes.categories || []
      const hasUncategorized = cats.some((c: Category) => c.name === '미분류')
      if (!hasUncategorized) {
        setCategories([{ id: 0, name: '미분류', keywords: '' }, ...cats])
      } else {
        setCategories(cats)
      }
      
      // 수입 카테고리에 '미분류' 추가
      const incCats = incCatRes.income_categories || []
      const hasIncUncategorized = incCats.some((c: IncomeCategory) => c.name === '미분류')
      if (!hasIncUncategorized) {
        setIncomeCategories([{ id: 0, name: '미분류', display_order: 0 }, ...incCats])
      } else {
        setIncomeCategories(incCats)
      }
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
      const catId = tx.category_id || ''
      setCategoryId(catId)
      setOriginalCategoryId(catId)
      setIncomeCategoryId(tx.income_category_id || '')
      setMemo(tx.memo || '')
      setIsRecurring(tx.is_recurring === 1)
      setIsAutoRepeat(tx.is_auto_repeat === 1)
    } catch (err) {
      console.error('Failed to load transaction:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleCategoryChange = async (value: number | '') => {
    setCategoryId(value)
    
    const isNewOrChanged = !isEdit || (isEdit && value !== originalCategoryId)
    if (description && value && isNewOrChanged) {
      try {
        const result = await api.v2.saveCategoryMapping(description, Number(value))
        if (result.success) {
          if (tooltipTimer.current) clearTimeout(tooltipTimer.current)
          setMappingTooltip({ show: true, categoryName: result.category_name })
          tooltipTimer.current = setTimeout(() => {
            setMappingTooltip(null)
          }, 3000)
        }
      } catch (err) {
        console.error('Failed to save category mapping:', err)
      }
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
        is_auto_repeat: isAutoRepeat ? 1 : 0,
      }

      if (isEdit && id) {
        await api.v2.updateTransaction(Number(id), data)
      } else {
        await api.v2.createTransaction(data)
      }
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
  
  // SMS 붙여넣기 처리
  const handleParseSMS = async () => {
    if (!smsText.trim()) return
    try {
      const result = await api.parseSMS(smsText)
      if (result.success && result.transaction) {
        const tx = result.transaction
        
        // 거래 유형 설정 (income/expense)
        if (tx.tx_type === 'income') {
          setTxType('income')
        } else {
          setTxType('expense')
        }
        
        // 자산 자동 매칭
        if (tx.asset_type_id) {
          // 백엔드에서 자산 ID를 반환한 경우
          if (tx.tx_type === 'income') {
            // 수입의 경우 가상 ID (입금: -1, 현금: -2)
            setAssetTypeId(tx.asset_type_id)
          } else {
            // 지출의 경우 DB 자산 ID
            setAssetTypeId(tx.asset_type_id)
          }
        } else if (tx.card_name) {
          // 카드 이름으로 자산 찾기 (fallback)
          const matchedAsset = expenseAssets.find(a => 
            a.name.includes(tx.card_name) || tx.card_name.includes(a.name)
          )
          if (matchedAsset) {
            setAssetTypeId(matchedAsset.id)
          }
        }
        
        setAmount(new Intl.NumberFormat('ko-KR').format(tx.amount))
        setDescription(tx.description || '')
        
        // 날짜 설정 (백엔드에서 yyyy-MM-dd 형식으로 반환)
        if (tx.date) {
          // 이미 yyyy-MM-dd 형식이면 그대로 사용, MM/dd 형식이면 변환
          if (tx.date.includes('-') && tx.date.length === 10) {
            setDate(tx.date)
          } else {
            const year = new Date().getFullYear()
            const dateStr = `${year}-${tx.date.replace('/', '-').padStart(5, '0')}`
            setDate(dateStr)
          }
        }
        
        setShowSmsModal(false)
        setSmsText('')
      } else {
        alert('문자 분석에 실패했습니다')
      }
    } catch (err) {
      console.error('Failed to parse SMS:', err)
      alert('문자 분석에 실패했습니다')
    }
  }

  const formatAmountInput = (value: string) => {
    const num = value.replace(/[^0-9]/g, '')
    if (!num) return ''
    return new Intl.NumberFormat('ko-KR').format(parseInt(num))
  }

  const currentAssets = txType === 'expense' ? expenseAssets : incomeAssets
  const currentCategories = txType === 'expense' ? categories : incomeCategories
  const selectedAssetName = currentAssets.find(a => a.id === assetTypeId)?.name || '선택해주세요'
  const selectedCategoryName = txType === 'expense' 
    ? categories.find(c => c.id === categoryId)?.name || '선택해주세요'
    : incomeCategories.find(c => c.id === incomeCategoryId)?.name || '선택해주세요'

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
        <div className={styles.headerActions}>
          {!isEdit && (
            <button className={styles.smsBtn} onClick={() => setShowSmsModal(true)}>
              <ClipboardPaste size={20} />
            </button>
          )}
          {isEdit && (
            <button className={styles.deleteBtn} onClick={handleDelete}>
              <Trash2 size={20} />
            </button>
          )}
        </div>
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
          onClick={() => { setTxType('income'); setAssetTypeId(-1); setIncomeCategoryId('') }}
        >
          <Plus size={18} /> 수입
        </button>
      </div>

      <div className={styles.form}>
        <div className={styles.field}>
          <span className={styles.label}>자산</span>
          <div className={styles.assetRow}>
            <button 
              className={styles.selectBtn}
              onClick={() => setShowAssetModal(true)}
            >
              <span className={assetTypeId ? styles.selected : styles.placeholder}>
                {selectedAssetName}
              </span>
              <span className={styles.chevron}>▼</span>
            </button>
            <label className={styles.recurringCheck}>
              <input 
                type="checkbox" 
                checked={isRecurring} 
                onChange={(e) => {
                  setIsRecurring(e.target.checked)
                  if (!e.target.checked) {
                    setIsAutoRepeat(false)
                    setShowRepeatTooltip(false)
                  }
                }} 
              />
              <span>고정{txType === 'expense' ? '지출' : '수입'}</span>
            </label>
          </div>
        </div>

        {/* 반복 옵션 - 고정 체크 시 표시 */}
        {isRecurring && (
          <div className={styles.repeatSection}>
            <label className={styles.repeatCheck}>
              <input 
                type="checkbox" 
                checked={isAutoRepeat} 
                onChange={(e) => {
                  setIsAutoRepeat(e.target.checked)
                  if (e.target.checked) {
                    setShowRepeatTooltip(true)
                    setTimeout(() => setShowRepeatTooltip(false), 5000)
                  }
                }} 
              />
              <span>매달 자동 반복</span>
            </label>
            {showRepeatTooltip && (
              <div className={styles.repeatTooltip}>
                <p>⚠️ 문자 자동등록 기능 사용 시 중복 등록될 수 있습니다.</p>
                <p>🔄 매달 이 날짜에 자동으로 내역이 등록됩니다.</p>
                <p>📝 내역 편집에서 언제든 반복을 해제할 수 있습니다.</p>
              </div>
            )}
          </div>
        )}

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
          <div className={styles.categoryWrapper}>
            <button 
              className={styles.selectBtn}
              onClick={() => setShowCategoryModal(true)}
            >
              <span className={(txType === 'expense' ? categoryId : incomeCategoryId) ? styles.selected : styles.placeholder}>
                {selectedCategoryName}
              </span>
              <span className={styles.chevron}>▼</span>
            </button>
            {mappingTooltip && mappingTooltip.show && (
              <div className={styles.mappingTooltip}>
                <Sparkles size={14} />
                <span>다음부터 <strong>{mappingTooltip.categoryName}</strong>으로 자동 입력됩니다</span>
              </div>
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
        disabled={submitting || !assetTypeId || !amount || !description || !date}
      >
        {submitting ? '저장 중...' : (isEdit ? '저장' : (txType === 'expense' ? '지출 등록' : '수입 등록'))}
      </button>

      {/* Asset Modal */}
      {showAssetModal && (
        <>
          <div className={styles.overlay} onClick={() => setShowAssetModal(false)} />
          <div className={styles.listModal}>
            <div className={styles.modalHeader}>
              <h3>자산 선택</h3>
              <button className={styles.closeBtn} onClick={() => setShowAssetModal(false)}>
                <X size={20} />
              </button>
            </div>
            <div className={styles.itemList}>
              {currentAssets.map(a => (
                <div 
                  key={a.id} 
                  className={`${styles.listItem} ${assetTypeId === a.id ? styles.active : ''}`}
                  onClick={() => {
                    setAssetTypeId(a.id)
                    setShowAssetModal(false)
                  }}
                >
                  <span className={styles.itemName}>{a.name}</span>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Category Modal */}
      {showCategoryModal && (
        <>
          <div className={styles.overlay} onClick={() => setShowCategoryModal(false)} />
          <div className={styles.listModal}>
            <div className={styles.modalHeader}>
              <h3>카테고리 선택</h3>
              <button className={styles.closeBtn} onClick={() => setShowCategoryModal(false)}>
                <X size={20} />
              </button>
            </div>
            <div className={styles.itemList}>
              {currentCategories.map(c => (
                <div 
                  key={c.id} 
                  className={`${styles.listItem} ${(txType === 'expense' ? categoryId : incomeCategoryId) === c.id ? styles.active : ''}`}
                  onClick={() => {
                    if (txType === 'expense') {
                      handleCategoryChange(c.id)
                    } else {
                      setIncomeCategoryId(c.id)
                    }
                    setShowCategoryModal(false)
                  }}
                >
                  <span className={styles.itemName}>{c.name}</span>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
      
      {/* SMS Modal */}
      {showSmsModal && (
        <>
          <div className={styles.overlay} onClick={() => setShowSmsModal(false)} />
          <div className={styles.smsModal}>
            <div className={styles.modalHeader}>
              <h3>문자 붙여넣기</h3>
              <button className={styles.closeBtn} onClick={() => setShowSmsModal(false)}>
                <X size={20} />
              </button>
            </div>
            <div className={styles.smsContent}>
              <textarea
                className={styles.smsTextarea}
                placeholder="카드 결제 문자를 붙여넣으세요"
                value={smsText}
                onChange={(e) => setSmsText(e.target.value)}
                rows={5}
              />
              <button 
                className={styles.parseBtn}
                onClick={handleParseSMS}
                disabled={!smsText.trim()}
              >
                분석하기
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
