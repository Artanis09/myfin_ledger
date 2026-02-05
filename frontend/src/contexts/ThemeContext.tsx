import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'

type Theme = 'light' | 'dark'

interface BillingColors {
  light: string
  dark: string
}

interface ThemeContextType {
  theme: Theme
  toggleTheme: () => void
  billingColors: BillingColors
  setBillingColor: (mode: 'light' | 'dark', color: string) => void
  showMemo: boolean
  setShowMemo: (show: boolean) => void
}

const defaultBillingColors: BillingColors = {
  light: '#ff9500',
  dark: '#0a84ff'
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('theme') as Theme
    if (saved) return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  const [billingColors, setBillingColors] = useState<BillingColors>(() => {
    const saved = localStorage.getItem('billingColors')
    if (saved) {
      try {
        return JSON.parse(saved)
      } catch {
        return defaultBillingColors
      }
    }
    return defaultBillingColors
  })

  const [showMemo, setShowMemoState] = useState<boolean>(() => {
    const saved = localStorage.getItem('showMemo')
    return saved !== 'false' // 기본값 true
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }, [theme])

  useEffect(() => {
    localStorage.setItem('billingColors', JSON.stringify(billingColors))
    // CSS 변수로 적용
    const currentColor = theme === 'dark' ? billingColors.dark : billingColors.light
    document.documentElement.style.setProperty('--billing-gradient-start', currentColor)
  }, [billingColors, theme])

  const toggleTheme = () => {
    setTheme(prev => prev === 'light' ? 'dark' : 'light')
  }

  const setBillingColor = (mode: 'light' | 'dark', color: string) => {
    setBillingColors(prev => ({ ...prev, [mode]: color }))
  }

  const setShowMemo = (show: boolean) => {
    setShowMemoState(show)
    localStorage.setItem('showMemo', String(show))
  }

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, billingColors, setBillingColor, showMemo, setShowMemo }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) throw new Error('useTheme must be used within ThemeProvider')
  return context
}
