import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Skeleton } from '@/components/ui/skeleton'
import { QuestionLadderCard } from '@/components/QuestionLadderCard'
import { ApiError, api, type ReviewQueueItem } from '@/lib/api'

const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

export default function ReviewPage() {
  const [queue, setQueue] = useState<ReviewQueueItem[] | null>(null)
  const [index, setIndex] = useState(0)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .getReviewQueue()
      .then((res) => setQueue(res.questions))
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить очередь повторения')
      })
  }, [])

  if (error) {
    return <p className="text-sm text-[var(--err)]">{error}</p>
  }

  if (queue === null) {
    return (
      <div className="mx-auto max-w-[560px] space-y-4">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-24 w-full" />
      </div>
    )
  }

  const current = queue[index]

  if (!current) {
    return (
      <div className="mx-auto flex max-w-[420px] flex-col items-center gap-2.5 rounded-[var(--r-md)] border border-dashed border-[#D8D3C8] px-7 py-10 text-center">
        <div style={{ fontFamily: FONT_CODE }} className="text-[20px] font-semibold text-[#C9C3B7]">
          [ ]
        </div>
        <p style={{ fontFamily: FONT_UI }} className="text-[14px] font-medium text-[var(--ink)]">
          На сегодня повторений нет
        </p>
        <p
          style={{ fontFamily: FONT_UI }}
          className="max-w-[300px] text-[12.5px] leading-[1.5] text-[var(--ink-3)]"
        >
          Очередь повторения пуста. Можно пройти новую главу учебника.
        </p>
        <Link
          to="/learn"
          style={{ fontFamily: FONT_UI }}
          className="mt-2 rounded-full border border-[var(--accent)] px-4 py-2 text-[13px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)]"
        >
          К учебнику
        </Link>
      </div>
    )
  }

  const total = queue.length
  const pct = Math.round(((index + 1) / total) * 100)
  const remainingMinutes = Math.max(1, Math.round((total - index) * 1.5))

  return (
    <div className="mx-auto max-w-[760px] pb-16">
      {/* Минимальный хром режима повторения (1n) */}
      <div className="mb-10 flex items-center gap-4 border-b border-[var(--hairline)] pb-4">
        <Link
          to="/"
          style={{ fontFamily: FONT_UI }}
          className="text-[13px] font-medium text-[var(--ink-2)] transition-colors hover:text-[var(--ink)]"
        >
          ✕ Завершить
        </Link>
        <div className="mx-auto flex w-full max-w-[320px] flex-1 items-center gap-3">
          <div className="h-1 flex-1 overflow-hidden rounded-[2px] bg-[var(--hairline)]">
            <div className="h-full bg-[var(--accent)]" style={{ width: `${pct}%` }} />
          </div>
          <span
            style={{ fontFamily: FONT_CODE }}
            className="shrink-0 text-[11px] text-[var(--ink-3)] tabular-nums"
          >
            {index + 1} / {total}
          </span>
        </div>
        <span style={{ fontFamily: FONT_CODE }} className="text-[11px] text-[var(--ink-3)]">
          ~{remainingMinutes} мин
        </span>
      </div>

      <QuestionLadderCard
        key={current.slug}
        slug={current.slug}
        mode="review"
        onGraded={() => setIndex((i) => i + 1)}
      />
    </div>
  )
}
