import { Outlet, NavLink } from 'react-router-dom'
import { Home, Receipt, BarChart3, Settings } from 'lucide-react'
import styles from './Layout.module.css'

const navItems = [
  { path: '/', icon: Home, label: '홈' },
  { path: '/transactions', icon: Receipt, label: '카드' },
  { path: '/statistics', icon: BarChart3, label: '통계' },
  { path: '/settings', icon: Settings, label: '설정' },
]

export default function Layout() {
  return (
    <div className={styles.container}>
      <main className={styles.main}>
        <Outlet />
      </main>
      
      <nav className={styles.bottomNav}>
        {navItems.map(({ path, icon: Icon, label }) => (
          <NavLink
            key={path}
            to={path}
            className={({ isActive }) => 
              `${styles.navItem} ${isActive ? styles.active : ''}`
            }
          >
            <Icon size={24} />
            <span>{label}</span>
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
