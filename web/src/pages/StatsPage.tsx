import { useEffect, useState } from 'react'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { Heatmap } from '@/components/Heatmap'
import { ApiError, api, type Stats } from '@/lib/api'
import { plural } from '@/lib/format'

interface StatNumber {
  value: number
  label: string
}

export default function StatsPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .getStats()
      .then(setStats)
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить статистику')
      })
  }, [])

  if (error) {
    return <p className="mx-auto max-w-[820px] text-sm text-err">{error}</p>
  }

  if (!stats) {
    return (
      <div className="mx-auto grid max-w-[820px] grid-cols-2 gap-px sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
    )
  }

  const learned = stats.by_section.reduce((sum, s) => sum + s.done, 0)
  const streakLabel =
    stats.streak_record != null
      ? `${plural(stats.streak_days, ['день', 'дня', 'дней'])} серия · рекорд ${stats.streak_record}`
      : `${plural(stats.streak_days, ['день', 'дня', 'дней'])} серия`

  const numbers: StatNumber[] = [
    { value: learned, label: 'вопросов изучено' },
    { value: stats.reviewed, label: 'повторений сделано' },
  ]
  if (stats.coding) numbers.push({ value: stats.coding.solved, label: 'задач решено' })
  numbers.push({ value: stats.streak_days, label: streakLabel })

  const hasActivity = Array.isArray(stats.activity) && stats.activity.length > 0

  return (
    <div className="mx-auto flex max-w-[820px] flex-col gap-8">
      <h1 className="text-[24px] font-semibold text-ink">Статистика</h1>

      <div
        className="grid grid-cols-2 gap-px overflow-hidden border-y border-hairline bg-hairline sm:grid-cols-4"
      >
        {numbers.map((n) => (
          <div key={n.label} className="flex flex-col gap-1 bg-bg px-5 py-5">
            <span className="font-mono text-[30px] font-semibold leading-none text-ink">{n.value}</span>
            <span className="text-xs text-ink-2">{n.label}</span>
          </div>
        ))}
      </div>

      {hasActivity && (
        <div className="flex flex-col gap-3.5">
          <h2 className="text-[15px] font-semibold text-ink">Последние 12 недель</h2>
          <Heatmap activity={stats.activity!} />
        </div>
      )}

      <div>
        <h2 className="mb-2.5 text-[15px] font-semibold text-ink">Прогресс по секциям</h2>
        {stats.by_section.length === 0 ? (
          <p className="text-sm text-ink-3">Нет данных по секциям.</p>
        ) : (
          <div>
            {stats.by_section.map((s) => {
              const pct = s.total > 0 ? Math.round((s.done / s.total) * 100) : 0
              return (
                <div
                  key={s.id}
                  className="flex items-center gap-4 border-t border-hairline py-3 last:border-b"
                >
                  <span className="w-[130px] shrink-0 truncate text-sm text-ink sm:w-[220px]">
                    {s.title}
                  </span>
                  <Progress value={pct} className="flex-1" />
                  <span className="w-14 shrink-0 text-right font-mono text-[11px] text-ink-3">
                    {s.done} / {s.total}
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
