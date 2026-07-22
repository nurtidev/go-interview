export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  }).format(date)
}

/** Русская форма слова по числу: plural(5, ['день','дня','дней']) → 'дней'. */
export function plural(n: number, forms: [string, string, string]): string {
  const abs = Math.abs(n) % 100
  const n1 = abs % 10
  if (abs > 10 && abs < 20) return forms[2]
  if (n1 > 1 && n1 < 5) return forms[1]
  if (n1 === 1) return forms[0]
  return forms[2]
}

/** Целое число дней от «сегодня» до даты (может быть отрицательным). null — если дата невалидна. */
export function daysUntil(iso: string | null | undefined): number | null {
  if (!iso) return null
  const target = new Date(iso)
  if (Number.isNaN(target.getTime())) return null
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const diffMs = startOfDay(target) - startOfDay(new Date())
  return Math.round(diffMs / 86_400_000)
}

/** «Добрый день» и т.п. по локальному времени. */
export function greeting(): string {
  const h = new Date().getHours()
  if (h < 6) return 'Доброй ночи'
  if (h < 12) return 'Доброе утро'
  if (h < 18) return 'Добрый день'
  return 'Добрый вечер'
}
