// Вспомогательные функции лайвкодинга: ключи localStorage, подписи статусов,
// сравнение результата SQL с эталоном. Всё здесь — чистая логика без wasm/сети,
// поэтому модуль безопасно импортировать статически.

import type { CodingStatus, Difficulty } from '@/lib/api'

// ---------- ключи localStorage ----------

export const timerKey = (slug: string): string => `goprep:timer:${slug}`
export const codeKey = (slug: string): string => `goprep:code:${slug}`
export const hintsKey = (slug: string): string => `goprep:hints:${slug}`

// ---------- черновик кода ----------

export function readDraft(slug: string): string | null {
  try {
    return localStorage.getItem(codeKey(slug))
  } catch {
    return null
  }
}

export function writeDraft(slug: string, code: string): void {
  try {
    localStorage.setItem(codeKey(slug), code)
  } catch {
    // приватный режим / переполнение — тихо игнорируем
  }
}

// ---------- прогресс подсказок ----------

export function readOpenedHints(slug: string, max: number): number {
  try {
    const raw = localStorage.getItem(hintsKey(slug))
    if (!raw) return 0
    const n = Number(raw)
    if (Number.isNaN(n) || n < 0) return 0
    return Math.min(n, max)
  } catch {
    return 0
  }
}

export function writeOpenedHints(slug: string, n: number): void {
  try {
    localStorage.setItem(hintsKey(slug), String(n))
  } catch {
    // игнорируем
  }
}

// ---------- таймер попытки 25/5 ----------

export function attemptTotalSeconds(difficulty: Difficulty): number {
  return difficulty === 'staff' ? 45 * 60 : 25 * 60
}

/** Стартовый timestamp попытки (мс). Устанавливается при первом открытии задачи. */
export function readOrInitTimerStart(slug: string): number {
  try {
    const raw = localStorage.getItem(timerKey(slug))
    const parsed = raw ? Number(raw) : NaN
    if (raw && !Number.isNaN(parsed)) return parsed
  } catch {
    // упадём в ветку инициализации
  }
  const start = Date.now()
  try {
    localStorage.setItem(timerKey(slug), String(start))
  } catch {
    // игнорируем
  }
  return start
}

// ---------- подписи статуса ----------

export const CODING_STATUS_LABELS: Record<CodingStatus, string> = {
  new: 'новая',
  attempted: 'в работе',
  solved: 'решена',
}

// Пилюли-статусы (спека 1b): «не начато» — контур; «в работе» — info-тинт;
// «решена» — ok-тинт.
export const CODING_STATUS_STYLES: Record<CodingStatus, string> = {
  new: 'border border-[var(--hairline)] text-[var(--ink-3)]',
  attempted: 'bg-[var(--info-soft)] text-[var(--info)]',
  solved: 'bg-[var(--ok-soft)] text-[var(--ok)]',
}

// ---------- разбор вывода go test ----------

/** Подсчёт пройденных/упавших тестов из вывода go test. */
export function countTestResults(output: string): { passed: number; failed: number } {
  let passed = 0
  let failed = 0
  for (const line of output.split('\n')) {
    if (/(^|---\s+)PASS\b/.test(line) && /Test/.test(line)) passed++
    else if (/(^|---\s+)FAIL\b/.test(line) && /Test/.test(line)) failed++
  }
  return { passed, failed }
}

// ---------- сравнение результата SQL с эталоном ----------

export interface RowsDiff {
  matched: boolean
  /** Строки эталона, которых не хватает в ответе пользователя (мультимножество). */
  missing: string[][]
  /** Лишние строки в ответе пользователя. */
  extra: string[][]
}

function rowKey(row: string[]): string {
  return JSON.stringify(row)
}

function countRows(rows: string[][]): Map<string, number> {
  const map = new Map<string, number>()
  for (const row of rows) {
    const key = rowKey(row)
    map.set(key, (map.get(key) ?? 0) + 1)
  }
  return map
}

/**
 * Сравнивает результат пользователя с эталоном.
 * order_matters=false — как мультимножества строк-кортежей; true — по порядку.
 * missing/extra всегда считаются по мультимножеству (для diff-подсказки).
 */
export function compareRows(
  actual: string[][],
  expected: string[][],
  orderMatters: boolean,
): RowsDiff {
  const actualCounts = countRows(actual)
  const expectedCounts = countRows(expected)

  const missing: string[][] = []
  for (const [key, count] of expectedCounts) {
    const have = actualCounts.get(key) ?? 0
    for (let i = 0; i < count - have; i++) missing.push(JSON.parse(key) as string[])
  }

  const extra: string[][] = []
  for (const [key, count] of actualCounts) {
    const need = expectedCounts.get(key) ?? 0
    for (let i = 0; i < count - need; i++) extra.push(JSON.parse(key) as string[])
  }

  const sameMultiset = missing.length === 0 && extra.length === 0
  const matched = orderMatters
    ? actual.length === expected.length &&
      actual.every((row, i) => rowKey(row) === rowKey(expected[i]))
    : sameMultiset

  return { matched, missing, extra }
}
