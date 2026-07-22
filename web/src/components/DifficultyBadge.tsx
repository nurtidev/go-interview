import { cn } from '@/lib/utils'
import type { Difficulty } from '@/lib/api'

const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'

const LABELS: Record<Difficulty, string> = {
  middle: 'middle',
  senior: 'senior',
  staff: 'staff',
}

// Бейджи сложности (спека 1b): заливка-тинт + семантический цвет текста.
// middle — приглушённый ink-2; senior — полный ink; staff — плюм.
const STYLES: Record<Difficulty, string> = {
  middle: 'bg-[var(--accent-soft)] text-[var(--ink-2)]',
  senior: 'bg-[var(--accent-soft)] text-[var(--ink)]',
  staff: 'bg-[var(--staff-soft)] text-[var(--staff)]',
}

export function DifficultyBadge({ difficulty }: { difficulty: Difficulty }) {
  return (
    <span
      style={{ fontFamily: FONT_UI }}
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-[3px] text-[11.5px] font-medium',
        STYLES[difficulty],
      )}
    >
      {LABELS[difficulty]}
    </span>
  )
}
