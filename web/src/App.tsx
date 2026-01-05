import { useState, useEffect } from 'react'
import Login from './pages/Login'
import Register from './pages/Register'
import Dashboard from './pages/Dashboard'

interface UserState {
  username: string
  isAdmin: boolean
}

function App() {
  const [page, setPage] = useState<'login' | 'register'>('login')
  const [user, setUser] = useState<UserState | null>(null)

  useEffect(() => {
    const token = localStorage.getItem('token')
    const username = localStorage.getItem('username')
    const isAdmin = localStorage.getItem('isAdmin') === 'true'
    if (token && username) {
      setUser({ username, isAdmin })
    }
  }, [])

  const handleSuccess = (username: string, token?: string, isAdmin?: boolean) => {
    if (token) {
      localStorage.setItem('token', token)
      localStorage.setItem('username', username)
      localStorage.setItem('isAdmin', String(isAdmin || false))
    }
    setUser({ username, isAdmin: isAdmin || false })
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('isAdmin')
    setUser(null)
    setPage('login')
  }

  if (user) {
    return <Dashboard username={user.username} isAdmin={user.isAdmin} onLogout={handleLogout} />
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100">
      {page === 'login' ? (
        <Login onSwitch={() => setPage('register')} onSuccess={handleSuccess} />
      ) : (
        <Register onSwitch={() => setPage('login')} onSuccess={handleSuccess} />
      )}
    </div>
  )
}

export default App
