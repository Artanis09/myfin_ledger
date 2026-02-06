const API_BASE = '/api'

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export const api = {
  // Dashboard
  getDashboard: (year?: number, month?: number) => {
    const params = new URLSearchParams()
    if (year) params.set('year', year.toString())
    if (month) params.set('month', month.toString())
    const query = params.toString() ? `?${params.toString()}` : ''
    return fetchJSON<any>(`/dashboard${query}`)
  },
  
  // Transactions
  getTransactions: (year?: number, month?: number) => {
    const params = new URLSearchParams()
    if (year) params.set('year', year.toString())
    if (month) params.set('month', month.toString())
    const query = params.toString() ? `?${params.toString()}` : ''
    return fetchJSON<any>(`/transactions${query}`)
  },
  getTransaction: (id: number) => fetchJSON<any>(`/transactions/${id}`),
  createTransaction: (data: any) => fetchJSON<any>('/transactions', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateTransaction: (id: number, data: any) => fetchJSON<any>(`/transactions/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteTransaction: (id: number) => fetchJSON<any>(`/transactions/${id}`, {
    method: 'DELETE',
  }),
  
  // Cards
  getCards: () => fetchJSON<any>('/cards'),
  createCard: (data: any) => fetchJSON<any>('/cards', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateCard: (id: number, data: any) => fetchJSON<any>(`/cards/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteCard: (id: number) => fetchJSON<any>(`/cards/${id}`, {
    method: 'DELETE',
  }),
  
  // Categories
  getCategories: () => fetchJSON<any>('/categories'),
  createCategory: (data: any) => fetchJSON<any>('/categories', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateCategory: (id: number, data: any) => fetchJSON<any>(`/categories/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteCategory: (id: number) => fetchJSON<any>(`/categories/${id}`, {
    method: 'DELETE',
  }),
  
  // Statistics
  getStatistics: (year?: number, month?: number) => {
    const params = new URLSearchParams()
    if (year) params.set('year', year.toString())
    if (month) params.set('month', month.toString())
    const query = params.toString() ? `?${params.toString()}` : ''
    return fetchJSON<any>(`/statistics${query}`)
  },
  
  // SMS Parsing
  parseSMS: (text: string) => fetchJSON<any>('/parse-sms', {
    method: 'POST',
    body: JSON.stringify({ text }),
  }),
  
  // Image Parsing
  parseImage: async (file: File) => {
    const formData = new FormData()
    formData.append('image', file)
    const res = await fetch(`${API_BASE}/parse-image`, {
      method: 'POST',
      body: formData,
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },
  
  // Bulk create transactions
  createTransactionsBulk: (transactions: any[]) => fetchJSON<any>('/transactions/bulk', {
    method: 'POST',
    body: JSON.stringify({ transactions }),
  }),
  
  // Weekly Stats
  getWeeklyStats: (year: number, month: number) => 
    fetchJSON<any>(`/weekly-stats?year=${year}&month=${month}`),
  
  // Goals
  getGoal: (year: number, month: number) => 
    fetchJSON<any>(`/goals?year=${year}&month=${month}`),
  setGoal: (year: number, month: number, targetAmount: number) => 
    fetchJSON<any>('/goals', {
      method: 'POST',
      body: JSON.stringify({ year, month, target_amount: targetAmount }),
    }),
  
  // iPhone Shortcut API
  getAPIKeys: () => fetchJSON<any[]>('/shortcut/keys'),
  createAPIKey: (name: string) => fetchJSON<{ api_key: string; name: string }>('/shortcut/keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  }),
  deleteAPIKey: (id: number) => fetchJSON<any>(`/shortcut/keys/${id}`, {
    method: 'DELETE',
  }),
  getSMSLogs: (limit = 50) => fetchJSON<any[]>(`/shortcut/logs?limit=${limit}`),
  deleteSMSLog: (id: number) => fetchJSON<any>(`/shortcut/logs/${id}`, {
    method: 'DELETE',
  }),
  deleteAllSMSLogs: () => fetchJSON<any>('/shortcut/logs', {
    method: 'DELETE',
  }),

  // ========== V2 API ==========
  v2: {
    // Settings
    getSettings: () => fetchJSON<any>('/v2/settings'),
    updateSettings: (data: Record<string, string>) => fetchJSON<any>('/v2/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

    // Asset Types
    getAssetTypes: (type?: 'expense' | 'income') => {
      const query = type ? `?type=${type}` : ''
      return fetchJSON<any>(`/v2/asset-types${query}`)
    },
    createAssetType: (data: any) => fetchJSON<any>('/v2/asset-types', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
    deleteAssetType: (id: number) => fetchJSON<any>(`/v2/asset-types/${id}`, {
      method: 'DELETE',
    }),

    // Income Categories
    getIncomeCategories: () => fetchJSON<any>('/v2/income-categories'),
    createIncomeCategory: (data: any) => fetchJSON<any>('/v2/income-categories', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
    updateIncomeCategory: (id: number, data: any) => fetchJSON<any>(`/v2/income-categories/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
    deleteIncomeCategory: (id: number) => fetchJSON<any>(`/v2/income-categories/${id}`, {
      method: 'DELETE',
    }),

    // Transactions
    getTransactions: (year: number, month: number, type?: 'expense' | 'income') => {
      const params = new URLSearchParams({ year: year.toString(), month: month.toString() })
      if (type) params.set('type', type)
      return fetchJSON<any>(`/v2/transactions?${params.toString()}`)
    },
    getTransaction: (id: number) => fetchJSON<any>(`/v2/transactions/${id}`),
    createTransaction: (data: any) => fetchJSON<any>('/v2/transactions', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
    updateTransaction: (id: number, data: any) => fetchJSON<any>(`/v2/transactions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
    deleteTransaction: (id: number) => fetchJSON<any>(`/v2/transactions/${id}`, {
      method: 'DELETE',
    }),

    // Dashboard
    getDashboard: (year: number, month: number) =>
      fetchJSON<any>(`/v2/dashboard?year=${year}&month=${month}`),

    // Statistics
    getStatisticsLedger: (year: number, month: number) =>
      fetchJSON<any>(`/v2/statistics/ledger?year=${year}&month=${month}`),
    getStatisticsCard: (year: number, month: number) =>
      fetchJSON<any>(`/v2/statistics/card?year=${year}&month=${month}`),

    // Recurring
    getRecurringSchedules: () => fetchJSON<any>('/v2/recurring'),
    deleteRecurringSchedule: (id: number) => fetchJSON<any>(`/v2/recurring/${id}`, {
      method: 'DELETE',
    }),

    // Category Mappings
    getCategoryMapping: (description: string) => 
      fetchJSON<{ category_id: number | null; category_name: string }>(
        `/v2/category-mapping?description=${encodeURIComponent(description)}`
      ),
    saveCategoryMapping: (description: string, categoryId: number) => 
      fetchJSON<{ success: boolean; category_id: number; category_name: string }>('/v2/category-mapping', {
        method: 'POST',
        body: JSON.stringify({ description, category_id: categoryId }),
      }),
    getAllCategoryMappings: () => fetchJSON<any>('/v2/category-mappings'),
    deleteCategoryMapping: (id: number) => fetchJSON<any>(`/v2/category-mapping/${id}`, {
      method: 'DELETE',
    }),

    // SMS Mapping Rules
    getSMSMappingRules: () => fetchJSON<any>('/v2/sms-mapping-rules'),
    createSMSMappingRule: (rule: { rule_type: string; keyword: string; tx_type?: string; priority: number; description?: string }) =>
      fetchJSON<any>('/v2/sms-mapping-rules', {
        method: 'POST',
        body: JSON.stringify(rule),
      }),
    deleteSMSMappingRule: (id: number) => fetchJSON<any>(`/v2/sms-mapping-rules/${id}`, {
      method: 'DELETE',
    }),
  },
}
