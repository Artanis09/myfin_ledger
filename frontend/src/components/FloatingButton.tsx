import { Plus } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import styles from './FloatingButton.module.css'

export default function FloatingButton() {
  const navigate = useNavigate()
  
  return (
    <button 
      className={styles.button}
      onClick={() => navigate('/transactions/new')}
    >
      <Plus size={28} />
    </button>
  )
}
