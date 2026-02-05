import { Plus } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import styles from './FloatingButton.module.css'

interface FloatingButtonProps {
  to?: string
}

export default function FloatingButton({ to = '/new' }: FloatingButtonProps) {
  const navigate = useNavigate()
  
  return (
    <button 
      className={styles.button}
      onClick={() => navigate(to)}
    >
      <Plus size={28} />
    </button>
  )
}
