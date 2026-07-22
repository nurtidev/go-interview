import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api'
import { useAuth } from '@/lib/auth'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await login(email, password)
      navigate('/', { replace: true })
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError('Неверный email или пароль')
      } else if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Не удалось войти. Попробуйте ещё раз.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-bg px-5 py-16">
      <div className="flex w-full max-w-[340px] flex-col gap-[22px]">
        <div className="flex flex-col items-center gap-2">
          <div className="text-[22px] font-semibold text-ink">
            GoPrep<span className="text-ink-3">_</span>
          </div>
          <p className="text-center text-[13.5px] leading-relaxed text-ink-2">
            Спортзал для собеседований Senior Go
          </p>
        </div>

        <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="email" className="text-[12.5px] text-ink-2">
              Email
            </Label>
            <Input
              id="email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="password" className="text-[12.5px] text-ink-2">
              Пароль
            </Label>
            <Input
              id="password"
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          {error && <p className="text-[12.5px] text-err">{error}</p>}
          <Button type="submit" size="lg" className="mt-1 w-full" disabled={loading}>
            {loading ? 'Входим…' : 'Войти'}
          </Button>
        </form>

        <p className="text-center text-[12.5px] text-ink-2">
          Нет аккаунта?{' '}
          <Link to="/register" className="font-medium text-ink transition-colors hover:text-accent-hover">
            Зарегистрироваться
          </Link>
        </p>
      </div>
    </div>
  )
}
