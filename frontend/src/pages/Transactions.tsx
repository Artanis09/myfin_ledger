import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { addMonths, subMonths } from 'date-fns'
import { Search, X } from 'lucide-react'
import Header from '../components/Header'
import TransactionList from '../components/TransactionList'
import FloatingButton from '../components/FloatingButton'
import { api } from '../api'
import type { Transaction, Card, Category } from '../types'
import styles from './Transactions.module.css'

type SortType = 'date-desc' | 'date-asc' | 'amount-desc' | 'amount-asc';

// 매달 1일이면 다음달(결제예정월)을 기본으로 표시
function getDefaultMonth() {
  const now = new Date()
  return addMonths(now, 1)
}

export default function Transactions() {
  const navigate = useNavigate()
  const [currentMonth, setCurrentMonth] = useState(getDefaultMonth)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [cards, setCards] = useState<Card[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [, setLoading] = useState(true)
  
  // 필터 상태
  const [showOnlyInstallment, setShowOnlyInstallment] = useState(false)
  const [sortBy, setSortBy] = useState<SortType>('date-desc')
  const [selectedCardId, setSelectedCardId] = useState<number | null>(null)
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [showSearch, setShowSearch] = useState(false)
  
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const year = currentMonth.getFullYear()
      const month = currentMonth.getMonth() + 1
      
      const [txData, cardData, catData] = await Promise.all([
        api.getTransactions(year, month),
        api.getCards(),
        api.getCategories()
      ])
      
      setTransactions(txData.transactions || [])
      setCards(cardData.cards || [])
      setCategories(catData.categories || [])
    } catch (err) {
      console.error('Failed to load:', err)
    } finally {
      setLoading(false)
    }
  }, [currentMonth])
  
  useEffect(() => {
    loadData()
  }, [loadData])
  
  // Auto refresh every 10 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      loadData()
    }, 10000)
    return () => clearInterval(interval)
  }, [loadData])
  
  const handleEdit = (id: number) => {
    navigate(`/transactions/${id}/edit`)
  }
  
  const handleDelete = async (ids: number[]) => {
    try {
      await Promise.all(ids.map(id => api.deleteTransaction(id)))
      loadData()
    } catch (err) {
      console.error('Failed to delete:', err)
      alert('삭제에 실패했습니다.')
    }
  }

  const filteredAndSortedTransactions = useMemo(() => {
    let result = [...transactions];

    // 카드 필터
    if (selectedCardId !== null) {
      result = result.filter(tx => tx.card_id === selectedCardId)
    }

    // 카테고리 필터
    if (selectedCategoryId !== null) {
      result = result.filter(tx => tx.category_id === selectedCategoryId)
    }

    // 할부 필터
    if (showOnlyInstallment) {
      result = result.filter(tx => tx.is_installment === 1);
    }

    // 검색 필터
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase()
      result = result.filter(tx => 
        tx.description.toLowerCase().includes(query) ||
        (tx.memo && tx.memo.toLowerCase().includes(query)) ||
        (tx.card_name && tx.card_name.toLowerCase().includes(query)) ||
        (tx.category_name && tx.category_name.toLowerCase().includes(query))
      )
    }

    // 정렬
    result.sort((a, b) => {
      switch (sortBy) {
        case 'date-desc':
          return new Date(b.transaction_date).getTime() - new Date(a.transaction_date).getTime();
        case 'date-asc':
          return new Date(a.transaction_date).getTime() - new Date(b.transaction_date).getTime();
        case 'amount-desc':
          return b.amount - a.amount;
        case 'amount-asc':
          return a.amount - b.amount;
        default:
          return 0;
      }
    });

    return result;
  }, [transactions, showOnlyInstallment, sortBy, selectedCardId, selectedCategoryId, searchQuery]);
  
  const totalExpense = filteredAndSortedTransactions.reduce((sum, tx) => {
    if (tx.is_cancelled === 1) {
      return sum - tx.amount
    }
    return sum + tx.amount
  }, 0)

  const clearFilters = () => {
    setSelectedCardId(null)
    setSelectedCategoryId(null)
    setShowOnlyInstallment(false)
    setSearchQuery('')
    setShowSearch(false)
  }

  const hasActiveFilters = selectedCardId !== null || selectedCategoryId !== null || showOnlyInstallment || searchQuery.trim()
  
  return (
    <div className={styles.page}>
      <Header 
        title="카드관리"
        month={currentMonth}
        onPrevMonth={() => setCurrentMonth(prev => subMonths(prev, 1))}
        onNextMonth={() => setCurrentMonth(prev => addMonths(prev, 1))}
      />
      
      {/* 검색 바 */}
      {showSearch && (
        <div className={styles.searchBar}>
          <Search size={18} className={styles.searchIcon} />
          <input
            type="text"
            placeholder="거래처, 내용, 메모 검색..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className={styles.searchInput}
            autoFocus
          />
          {searchQuery && (
            <button className={styles.clearSearchBtn} onClick={() => setSearchQuery('')}>
              <X size={16} />
            </button>
          )}
        </div>
      )}
      
      {/* 필터 바 */}
      <div className={styles.filterBar}>
        <div className={styles.filterGroup}>
          {/* 검색 버튼 */}
          <button 
            className={`${styles.filterBtn} ${showSearch || searchQuery ? styles.active : ''}`}
            onClick={() => setShowSearch(!showSearch)}
          >
            <Search size={14} />
          </button>
          
          {/* 카드 필터 */}
          <select
            className={`${styles.filterSelect} ${selectedCardId !== null ? styles.active : ''}`}
            value={selectedCardId ?? ''}
            onChange={e => setSelectedCardId(e.target.value ? Number(e.target.value) : null)}
          >
            <option value="">전체 카드</option>
            {cards.map(card => (
              <option key={card.id} value={card.id}>{card.name}</option>
            ))}
          </select>
          
          {/* 카테고리 필터 */}
          <select
            className={`${styles.filterSelect} ${selectedCategoryId !== null ? styles.active : ''}`}
            value={selectedCategoryId ?? ''}
            onChange={e => setSelectedCategoryId(e.target.value ? Number(e.target.value) : null)}
          >
            <option value="">전체 카테고리</option>
            {categories.map(cat => (
              <option key={cat.id} value={cat.id}>{cat.name}</option>
            ))}
          </select>
          
          {/* 할부만 보기 */}
          <button 
            className={`${styles.filterBtn} ${showOnlyInstallment ? styles.active : ''}`}
            onClick={() => setShowOnlyInstallment(!showOnlyInstallment)}
          >
            할부
          </button>
        </div>
        
        <div className={styles.sortGroup}>
          {/* 필터 초기화 */}
          {hasActiveFilters && (
            <button className={styles.clearFilterBtn} onClick={clearFilters}>
              초기화
            </button>
          )}
          
          {/* 정렬 */}
          <select 
            className={styles.sortSelect}
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as SortType)}
          >
            <option value="date-desc">날짜 최신순</option>
            <option value="date-asc">날짜 오래된순</option>
            <option value="amount-desc">금액 높은순</option>
            <option value="amount-asc">금액 낮은순</option>
          </select>
        </div>
      </div>

      <div className={styles.summary}>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>수입</span>
          <span className={styles.summaryIncome}>0원</span>
        </div>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>지출</span>
          <span className={styles.summaryExpense}>{totalExpense.toLocaleString()}원</span>
        </div>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>합계</span>
          <span className={styles.summaryTotal}>-{totalExpense.toLocaleString()}원</span>
        </div>
      </div>
      
      {filteredAndSortedTransactions.length > 0 ? (
        <TransactionList 
          transactions={filteredAndSortedTransactions}
          onEdit={handleEdit}
          onDelete={handleDelete}
          sortBy={sortBy}
          grouped={sortBy.startsWith('date')}
        />
      ) : (
        <div className={styles.empty}>
          <p>{hasActiveFilters ? '필터 결과가 없습니다' : '이 달 결제예정 내역이 없습니다'}</p>
        </div>
      )}
      
      <FloatingButton />
    </div>
  )
}
