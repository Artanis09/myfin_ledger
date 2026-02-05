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
  memo: string | null
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

// V2 Types
export interface AssetType {
  id: number
  name: string
  type: 'expense' | 'income'
  is_card: number
  card_id: number | null
  is_recurring: number
  display_order: number
  is_system: number
}

export interface IncomeCategory {
  id: number
  name: string
  display_order: number
}

export interface TransactionV2 {
  id: number
  tx_type: 'expense' | 'income'
  asset_type_id: number | null
  asset_type_name: string
  card_id: number | null
  category_id: number | null
  category_name: string
  income_category_id: number | null
  income_category_name: string
  transaction_date: string
  description: string
  amount: number
  is_installment: number
  installment_months: number | null
  is_cancelled: number
  memo: string | null
  recurring_schedule_id: number | null
}

export interface RecurringSchedule {
  id: number
  tx_type: 'expense' | 'income'
  asset_type_id: number
  asset_type_name: string
  category_id: number | null
  category_name: string
  income_category_id: number | null
  income_category_name: string
  description: string
  amount: number
  day_of_month: number
  memo: string | null
  is_active: number
  last_generated_date: string | null
}

export interface DashboardV2Data {
  year: number
  month: number
  period_start: string
  period_end: string
  total_income: number
  total_expense: number
  balance: number
  transactions: TransactionV2[]
  daily_summary: Record<string, Record<string, number>>
}

export interface LedgerSettings {
  ledger_period_start_day: string
  ledger_period_end_day: string
}

export interface LedgerStatistics {
  year: number
  month: number
  period_start: string
  period_end: string
  total_income: number
  total_expense: number
  balance: number
  by_category: CategoryStat[]
  by_asset: { asset_name: string; total_amount: number; count: number }[]
  by_income_category: CategoryStat[]
}
