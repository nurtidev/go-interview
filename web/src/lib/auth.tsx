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
  /** Открыта ли самостоятельная регистрация на этом инстансе (GET /api/config). */
  registrationEnabled: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => getStoredUser())
  // По умолчанию считаем регистрацию открытой: так ведёт себя приложение, пока
  // /api/config не ответил (или недоступен) — деградация без изменений.
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    function handleExpired() {
      setUser(null)
      navigate('/login', { replace: true })
    }
    window.addEventListener(AUTH_EXPIRED_EVENT, handleExpired)
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleExpired)
  }, [navigate])

  useEffect(() => {
    let cancelled = false
    api
      .getConfig()
      .then((cfg) => {
        if (!cancelled) setRegistrationEnabled(cfg.registration_enabled)
      })
      .catch(() => {
        // /api/config недоступен — оставляем значение по умолчанию (true).
      })
    return () => {
      cancelled = true
    }
  }, [])

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
    registrationEnabled,
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
