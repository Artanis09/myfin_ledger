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
}
