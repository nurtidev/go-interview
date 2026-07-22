import { useEffect, useState, type ComponentType, type ReactNode } from 'react'
import { Link, Outlet, useLocation } from 'react-router-dom'
import { useTheme } from 'next-themes'
import {
  SunIcon,
  MoonIcon,
  LogOutIcon,
  LayoutGridIcon,
  BookOpenIcon,
  ListChecksIcon,
  SquareCodeIcon,
  RepeatIcon,
  ChartColumnIcon,
} from 'lucide-react'
import { useAuth } from '@/lib/auth'
import { api, type Me, type Stats } from '@/lib/api'
import { plural } from '@/lib/format'
import { cn } from '@/lib/utils'

type IconType = ComponentType<{ className?: string }>

interface NavItem {
  to: string
  label: string
  icon: IconType
  /** Хэш-якорь для пунктов, ведущих на секцию дашборда. */
  hash?: string
  /** Какой счётчик «due» показывать рядом с пунктом. */
  badge?: 'review' | 'coding'
}

// Десктоп-навигация (6 пунктов). Мобильный таб-бар берёт первые 4 (mobile:true).
const NAV_ITEMS: (NavItem & { mobile?: boolean })[] = [
  { to: '/', label: 'Обзор', icon: LayoutGridIcon, mobile: true },
  { to: '/learn', label: 'Учебник', icon: BookOpenIcon, mobile: true },
  { to: '/', hash: '#sections', label: 'Вопросы', icon: ListChecksIcon, mobile: true },
  { to: '/code', label: 'Кодинг', icon: SquareCodeIcon, badge: 'coding', mobile: true },
  { to: '/review', label: 'Повторение', icon: RepeatIcon, badge: 'review' },
  { to: '/stats', label: 'Статистика', icon: ChartColumnIcon },
]

function isActive(item: NavItem, pathname: string, hash: string): boolean {
  if (item.hash) return pathname === item.to && hash === item.hash
  if (item.to === '/') return pathname === '/' && hash === ''
  return pathname === item.to || pathname.startsWith(`${item.to}/`)
}

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)
  useEffect(() => setMounted(true), [])
  const isDark = resolvedTheme === 'dark'

  return (
    <button
      type="button"
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
      title={isDark ? 'Светлая тема' : 'Тёмная тема'}
      aria-label="Переключить тему"
      className="inline-flex size-9 items-center justify-center rounded-full text-ink-2 transition-colors duration-150 ease-out hover:bg-accent-soft hover:text-ink"
    >
      {mounted ? (
        isDark ? <SunIcon className="size-[18px]" /> : <MoonIcon className="size-[18px]" />
      ) : (
        <span className="size-[18px]" />
      )}
    </button>
  )
}

function initialsFrom(me: Me | null, email: string | undefined): string {
  const name = me?.name?.trim()
  if (name) {
    const parts = name.split(/\s+/).filter(Boolean)
    return parts.slice(0, 2).map((p) => p[0]!.toUpperCase()).join('')
  }
  return (email?.[0] ?? '?').toUpperCase()
}

export function Layout({ children }: { children?: ReactNode }) {
  const { user, logout } = useAuth()
  const location = useLocation()
  const [stats, setStats] = useState<Stats | null>(null)
  const [me, setMe] = useState<Me | null>(null)

  useEffect(() => {
    api.getStats().then(setStats).catch(() => setStats(null))
  }, [location.pathname])

  useEffect(() => {
    api.getMe().then(setMe).catch(() => setMe(null))
  }, [])

  const dueReview = stats?.due_today ?? 0
  const dueCoding = stats?.coding?.due ?? 0
  const streak = stats?.streak_days ?? 0

  function badgeCount(item: NavItem): number {
    if (item.badge === 'review') return dueReview
    if (item.badge === 'coding') return dueCoding
    return 0
  }

  const mobileItems = NAV_ITEMS.filter((i) => i.mobile)

  return (
    <div className="min-h-svh bg-bg text-ink">
      <header className="sticky top-0 z-20 h-[56px] border-b border-hairline bg-bg/95 backdrop-blur-sm">
        <div className="mx-auto flex h-full max-w-[960px] items-center gap-8 px-5 sm:px-8">
          <Link to="/" className="text-[17px] font-semibold text-ink">
            GoPrep<span className="text-ink-3">_</span>
          </Link>

          <nav className="hidden items-center gap-6 text-[13.5px] font-medium min-[900px]:flex">
            {NAV_ITEMS.map((item) => {
              const active = isActive(item, location.pathname, location.hash)
              const count = badgeCount(item)
              return (
                <Link
                  key={item.label}
                  to={`${item.to}${item.hash ?? ''}`}
                  className={cn(
                    'relative -my-[18px] border-b-2 py-[18px] transition-colors duration-150 ease-out',
                    active
                      ? 'border-ink text-ink'
                      : 'border-transparent text-ink-2 hover:text-ink',
                  )}
                >
                  {item.label}
                  {count > 0 && <span className="ml-1.5 font-mono text-[11px] text-warn">{count}</span>}
                </Link>
              )
            })}
          </nav>

          <div className="ml-auto flex items-center gap-2 sm:gap-3">
            {streak > 0 && (
              <span className="hidden rounded-full bg-accent-soft px-3 py-1 font-mono text-xs text-ink sm:inline-block">
                серия · {streak} {plural(streak, ['день', 'дня', 'дней'])}
              </span>
            )}
            <ThemeToggle />
            <span
              className="hidden size-8 items-center justify-center rounded-full bg-accent-soft text-xs font-semibold text-ink-2 min-[900px]:inline-flex"
              title={me?.name ?? user?.email}
            >
              {initialsFrom(me, user?.email)}
            </span>
            <button
              type="button"
              onClick={logout}
              title="Выйти"
              aria-label="Выйти"
              className="inline-flex size-9 items-center justify-center rounded-full text-ink-2 transition-colors duration-150 ease-out hover:bg-accent-soft hover:text-ink"
            >
              <LogOutIcon className="size-[18px]" />
            </button>
          </div>
        </div>
      </header>

      <main className="px-5 pt-8 pb-24 sm:px-8 min-[900px]:pb-16">
        {children ?? <Outlet />}
      </main>

      {/* Мобильный нижний таб-бар (4 пункта), хиты ≥44px */}
      <nav className="fixed inset-x-0 bottom-0 z-20 grid grid-cols-4 border-t border-hairline bg-bg min-[900px]:hidden">
        {mobileItems.map((item) => {
          const active = isActive(item, location.pathname, location.hash)
          const count = badgeCount(item)
          const Icon = item.icon
          return (
            <Link
              key={item.label}
              to={`${item.to}${item.hash ?? ''}`}
              className={cn(
                'relative flex min-h-[56px] flex-col items-center justify-center gap-1 pt-2 transition-colors duration-150 ease-out',
                active ? 'text-ink' : 'text-ink-3',
              )}
              style={{ paddingBottom: 'calc(0.5rem + env(safe-area-inset-bottom))' }}
            >
              <Icon className="size-[22px]" />
              <span className="text-[10px] font-medium">{item.label}</span>
              {count > 0 && (
                <span className="absolute top-1.5 right-[calc(50%-18px)] flex size-4 items-center justify-center rounded-full bg-warn font-mono text-[9px] leading-none text-[var(--bg)]">
                  {count}
                </span>
              )}
            </Link>
          )
        })}
      </nav>
    </div>
  )
}
