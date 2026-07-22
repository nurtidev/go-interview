import { cn } from '@/lib/utils'
import type { QuestionStatus } from '@/lib/api'

const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'

const LABELS: Record<QuestionStatus, string> = {
  new: 'новый',
  learning: 'изучается',
  review: 'на повторении',
}

// Статусы (спека 1b): «не начато» — контур; «в процессе» — info-тинт;
// «к повторению» — warn-тинт.
const STYLES: Record<QuestionStatus, string> = {
  new: 'border border-[var(--hairline)] text-[var(--ink-3)]',
  learning: 'bg-[var(--info-soft)] text-[var(--info)]',
  review: 'bg-[var(--warn-soft)] text-[var(--warn)]',
}

export function StatusBadge({ status }: { status: QuestionStatus }) {
  return (
    <span
      style={{ fontFamily: FONT_UI }}
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-[3px] text-[11.5px] font-medium',
        STYLES[status],
      )}
    >
      {LABELS[status]}
    </span>
  )
}
