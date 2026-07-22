import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import type { Section } from '@/lib/api'

// Строка секции вопросов (макет 1c): название + подзаголовок, warn-бейдж «N к повторению»,
// прогресс 160px + счётчик mono. Иерархия — hairline-строками, hover — тинт --surface-2.
export function SectionCard({ section }: { section: Section }) {
  const pct = section.total > 0 ? Math.round((section.done / section.total) * 100) : 0
  const notStarted = section.done === 0 && section.due === 0

  return (
    <Link
      to={`/section/${section.id}`}
      className="group flex flex-col gap-3 border-t border-hairline py-3.5 transition-colors duration-150 ease-out hover:bg-surface-2 sm:flex-row sm:items-center sm:gap-4"
    >
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="text-[14.5px] font-medium text-ink group-hover:text-accent-hover">
          {section.title}
        </span>
        {section.description && (
          <span className="text-xs text-ink-3">{section.description}</span>
        )}
      </div>
      {section.due > 0 ? (
        <Badge variant="warn">{section.due} к повторению</Badge>
      ) : notStarted ? (
        <Badge variant="outline">не начато</Badge>
      ) : null}
      <div className="flex w-full flex-col gap-1.5 sm:w-40">
        <Progress value={pct} />
        <span className="font-mono text-[10.5px] text-ink-3">
          {section.done} / {section.total}
        </span>
      </div>
    </Link>
  )
}
