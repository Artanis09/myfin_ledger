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
  getDashboard: () => fetchJSON<any>('/dashboard'),
  
  // Transactions
  getTransactions: () => fetchJSON<any>('/transactions'),
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
  getStatistics: (start: string, end: string) => 
    fetchJSON<any>(`/statistics?start=${start}&end=${end}`),
  
  // SMS Parsing
  parseSMS: (text: string) => fetchJSON<any>('/parse-sms', {
    method: 'POST',
    body: JSON.stringify({ text }),
  }),
}
