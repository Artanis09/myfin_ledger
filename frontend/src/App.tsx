import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Transactions from './pages/Transactions'
import Statistics from './pages/Statistics'
import Cards from './pages/Cards'
import Categories from './pages/Categories'
import Settings from './pages/Settings'
import TransactionForm from './pages/TransactionForm'
import Shortcut from './pages/Shortcut'

function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="transactions" element={<Transactions />} />
        <Route path="transactions/new" element={<TransactionForm />} />
        <Route path="transactions/:id/edit" element={<TransactionForm />} />
        <Route path="statistics" element={<Statistics />} />
        <Route path="cards" element={<Cards />} />
        <Route path="categories" element={<Categories />} />
        <Route path="settings" element={<Settings />} />
        <Route path="shortcut" element={<Shortcut />} />
      </Route>
    </Routes>
  )
}

export default App
