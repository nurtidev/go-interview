import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AUTH_EXPIRED_EVENT,
  api,
  clearStoredAuth,
  getStoredUser,
  getToken,
  setStoredAuth,
  type User,
} from '@/lib/api'

interface AuthContextValue {
  user: User | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => getStoredUser())
  const navigate = useNavigate()

  useEffect(() => {
    function handleExpired() {
      setUser(null)
      navigate('/login', { replace: true })
    }
    window.addEventListener(AUTH_EXPIRED_EVENT, handleExpired)
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleExpired)
  }, [navigate])

  async function login(email: string, password: string) {
    const res = await api.login(email, password)
    setStoredAuth(res.token, res.user)
    setUser(res.user)
  }

  async function register(email: string, password: string) {
    const res = await api.register(email, password)
    setStoredAuth(res.token, res.user)
    setUser(res.user)
  }

  function logout() {
    clearStoredAuth()
    setUser(null)
    navigate('/login', { replace: true })
  }

  const value: AuthContextValue = {
    user,
    isAuthenticated: Boolean(user && getToken()),
    login,
    register,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth должен использоваться внутри AuthProvider')
  return ctx
}
