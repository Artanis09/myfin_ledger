import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Transactions from './pages/Transactions'
// import Statistics from './pages/Statistics'
import Cards from './pages/Cards'
import Categories from './pages/Categories'
import Settings from './pages/Settings'
import TransactionForm from './pages/TransactionForm'
import Shortcut from './pages/Shortcut'
import HomeV2 from './pages/HomeV2'
import TransactionFormV2 from './pages/TransactionFormV2'
import StatisticsV2 from './pages/StatisticsV2'

function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        {/* V2 Routes (New unified system) */}
        <Route index element={<HomeV2 />} />
        <Route path="new" element={<TransactionFormV2 />} />
        <Route path="edit/:id" element={<TransactionFormV2 />} />
        <Route path="statistics" element={<StatisticsV2 />} />
        
        {/* Legacy Routes */}
        <Route path="legacy" element={<Dashboard />} />
        <Route path="transactions" element={<Transactions />} />
        <Route path="transactions/new" element={<TransactionForm />} />
        <Route path="transactions/:id/edit" element={<TransactionForm />} />
        <Route path="cards" element={<Cards />} />
        <Route path="categories" element={<Categories />} />
        <Route path="settings" element={<Settings />} />
        <Route path="shortcut" element={<Shortcut />} />
      </Route>
    </Routes>
  )
}

export default App
