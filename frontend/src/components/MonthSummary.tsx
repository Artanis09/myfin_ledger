import styles from './MonthSummary.module.css'

interface MonthSummaryProps {
  income: number
  expense: number
}

export default function MonthSummary({ income, expense }: MonthSummaryProps) {
  const balance = income - expense
  const incomePercent = income > 0 ? 100 : 0
  const expensePercent = income > 0 ? Math.round((expense / income) * 100) : (expense > 0 ? 100 : 0)
  
  const formatMoney = (amount: number) => {
    return amount.toLocaleString() + '원'
  }
  
  return (
    <div className={styles.container}>
      <div className={styles.circles}>
        <div className={styles.circleWrapper}>
          <svg className={styles.circleSvg} viewBox="0 0 36 36">
            <path
              className={styles.circleBg}
              d="M18 2.0845
                a 15.9155 15.9155 0 0 1 0 31.831
                a 15.9155 15.9155 0 0 1 0 -31.831"
            />
            <path
              className={styles.circleIncome}
              strokeDasharray={`${incomePercent}, 100`}
              d="M18 2.0845
                a 15.9155 15.9155 0 0 1 0 31.831
                a 15.9155 15.9155 0 0 1 0 -31.831"
            />
          </svg>
          <div className={styles.circleText}>
            <span className={styles.percent}>{incomePercent}%</span>
          </div>
          <span className={styles.label}>수입</span>
        </div>
        
        <div className={styles.circleWrapper}>
          <svg className={styles.circleSvg} viewBox="0 0 36 36">
            <path
              className={styles.circleBg}
              d="M18 2.0845
                a 15.9155 15.9155 0 0 1 0 31.831
                a 15.9155 15.9155 0 0 1 0 -31.831"
            />
            <path
              className={styles.circleExpense}
              strokeDasharray={`${expensePercent}, 100`}
              d="M18 2.0845
                a 15.9155 15.9155 0 0 1 0 31.831
                a 15.9155 15.9155 0 0 1 0 -31.831"
            />
          </svg>
          <div className={styles.circleText}>
            <span className={styles.percent}>{expensePercent}%</span>
          </div>
          <span className={styles.label}>지출</span>
        </div>
        
        <div className={styles.balanceBox}>
          <span className={styles.balanceLabel}>합계</span>
          <span className={`${styles.balanceAmount} ${balance < 0 ? styles.negative : ''}`}>
            {formatMoney(balance)}
          </span>
        </div>
      </div>
      
      <div className={styles.amounts}>
        <div className={styles.amountItem}>
          <span className={styles.amountValue + ' ' + styles.income}>{formatMoney(income)}</span>
        </div>
        <div className={styles.amountItem}>
          <span className={styles.amountValue + ' ' + styles.expense}>{formatMoney(expense)}</span>
        </div>
      </div>
    </div>
  )
}
