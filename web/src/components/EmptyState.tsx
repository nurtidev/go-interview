import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  title: string
  description?: ReactNode
  /** Следующее действие — контурная pill-кнопка. Ссылка (to) или обработчик (onClick). */
  actionLabel?: string
  actionTo?: string
  onAction?: () => void
  glyph?: string
  className?: string
}

// Empty state (макет 1b): dashed-рамка, глиф [ ], заголовок, пояснение, контурная кнопка.
export function EmptyState({
  title,
  description,
  actionLabel,
  actionTo,
  onAction,
  glyph = '[ ]',
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center gap-2.5 rounded-[10px] border border-dashed border-field-border px-7 py-8 text-center',
        className,
      )}
    >
      <div className="font-mono text-xl font-semibold text-[var(--glyph-empty)]">{glyph}</div>
      <div className="text-sm font-medium text-ink">{title}</div>
      {description && (
        <div className="max-w-[300px] text-xs leading-relaxed text-ink-3">{description}</div>
      )}
      {actionLabel &&
        (actionTo ? (
          <Button asChild variant="outline" size="sm" className="mt-1">
            <Link to={actionTo}>{actionLabel}</Link>
          </Button>
        ) : (
          <Button variant="outline" size="sm" className="mt-1" onClick={onAction}>
            {actionLabel}
          </Button>
        ))}
    </div>
  )
}
