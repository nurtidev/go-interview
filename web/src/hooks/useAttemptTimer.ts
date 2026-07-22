import { useEffect, useRef, useState } from 'react'
import type { Difficulty } from '@/lib/api'
import { attemptTotalSeconds, readOrInitTimerStart } from '@/lib/coding'

const HINT_INTERVAL_SECONDS = 5 * 60

export interface AttemptTimerState {
  /** Фаза попытки: обратный отсчёт (focus) либо счёт «сверх» времени (over). */
  phase: 'focus' | 'over'
  /** Строка для пилюли: «24:37» в фокусе или «+03:12» сверх времени. */
  display: string
  /** Время вышло — пилюля переходит в янтарную фазу. */
  isTimeUp: boolean
  /** Номер подсказки (1-based), которую пора открыть, либо null. */
  nudgeHint: number | null
  /** Секунды до открытия следующей ещё не открытой подсказки (в фазе over), либо null. */
  nextHintInSeconds: number | null
}

function formatClock(totalSeconds: number): string {
  const seconds = Math.max(0, totalSeconds)
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`
}

/** «M:SS» без ведущего нуля минут — для подписи «подсказка через M:SS». */
export function formatHintCountdown(totalSeconds: number): string {
  const seconds = Math.max(0, totalSeconds)
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return `${minutes}:${String(rest).padStart(2, '0')}`
}

/**
 * Таймер попытки 25/5. Отсчёт вниз от 25:00 (45:00 для staff), персист старта
 * в localStorage — на повторных заходах продолжается. После нуля идёт счёт вверх
 * «+MM:SS»; каждые 5 минут разблокируется очередная ещё не открытая подсказка.
 */
export function useAttemptTimer(
  slug: string,
  difficulty: Difficulty,
  openedHints: number,
  hintsCount: number,
): AttemptTimerState {
  const total = attemptTotalSeconds(difficulty)
  const startRef = useRef<number>(0)
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    startRef.current = readOrInitTimerStart(slug)
    setNow(Date.now())
  }, [slug])

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [])

  const start = startRef.current || now
  const elapsed = Math.max(0, Math.floor((now - start) / 1000))
  const remaining = total - elapsed
  const isTimeUp = remaining <= 0
  const phase: 'focus' | 'over' = isTimeUp ? 'over' : 'focus'

  const display = isTimeUp ? `+${formatClock(elapsed - total)}` : formatClock(remaining)

  let nudgeHint: number | null = null
  let nextHintInSeconds: number | null = null
  if (isTimeUp && openedHints < hintsCount) {
    const nextHint = openedHints + 1 // 1-based
    const dueAtElapsed = total + (nextHint - 1) * HINT_INTERVAL_SECONDS
    if (elapsed >= dueAtElapsed) nudgeHint = nextHint
    else nextHintInSeconds = dueAtElapsed - elapsed
  }

  return { phase, display, isTimeUp, nudgeHint, nextHintInSeconds }
}
