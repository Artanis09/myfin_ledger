export interface Card {
  id: number
  name: string
  billing_start_day: number
  billing_end_day: number
}

export interface Category {
  id: number
  name: string
  keywords: string
}

export interface Transaction {
  id: number
  card_id: number
  category_id: number | null
  card_name: string
  category_name: string | null
  transaction_date: string
  description: string
  amount: number
  is_installment: number
  installment_months: number | null
  installment_current: number | null
  original_amount: number | null
  is_cancelled: number
}

export interface BillingPeriod {
  card_name: string
  card_id: number
  start_day: number
  end_day: number
  start_date: string
  end_date: string
  total: number
}

export interface DashboardData {
  total_this_month: number
  current_month: string
  billing_periods: BillingPeriod[]
  recent_transactions: Transaction[]
}

export interface CategoryStat {
  category_name: string
  total_amount: number
  count: number
}

export interface CardStat {
  card_name: string
  card_id: number
  total_amount: number
  count: number
}

export interface StatisticsData {
  by_category: CategoryStat[]
  by_card: CardStat[]
  total: number
}

export interface ParsedSMS {
  card_name: string
  amount: number
  description: string
  date: string
  time: string
  is_installment: boolean
  installment_months: number
  is_cancelled: boolean
}
