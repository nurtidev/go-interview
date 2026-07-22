import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Skeleton } from '@/components/ui/skeleton'
import { DifficultyBadge } from '@/components/DifficultyBadge'
import { ApiError, api, type CodingKind, type CodingTaskListItem } from '@/lib/api'
import { CODING_STATUS_LABELS, CODING_STATUS_STYLES } from '@/lib/coding'
import { cn } from '@/lib/utils'

const FONT_CONTENT = 'var(--font-content, Georgia, serif)'
const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

const GROUP_ORDER: CodingKind[] = ['go', 'sql']
const GROUP_LABELS: Record<CodingKind, string> = {
  go: 'Go',
  sql: 'SQL',
}

function TaskRow({ task }: { task: CodingTaskListItem }) {
  const showGiveUpHint = task.gave_up && task.status !== 'solved'

  return (
    <Link
      to={`/code/${task.slug}`}
      className="flex items-start justify-between gap-4 border-b border-[var(--hairline)] px-1.5 py-5 transition-colors hover:bg-[var(--accent-soft)]"
    >
      <div className="min-w-0 flex-1">
        <h3 style={{ fontFamily: FONT_CONTENT }} className="text-lg font-bold text-[var(--ink)]">
          {task.title}
        </h3>
        <div className="mt-1.5 flex flex-wrap items-center gap-2">
          <DifficultyBadge difficulty={task.difficulty} />
          {task.tags.length > 0 && (
            <span style={{ fontFamily: FONT_UI }} className="text-[13px] text-[var(--ink-3)]">
              {task.tags.map((t) => `#${t}`).join(' ')}
            </span>
          )}
          <span
            style={{ fontFamily: FONT_UI }}
            className={cn(
              'inline-flex items-center rounded-full px-2.5 py-[3px] text-[11.5px] font-medium',
              CODING_STATUS_STYLES[task.status],
            )}
          >
            {CODING_STATUS_LABELS[task.status]}
          </span>
          {showGiveUpHint && (
            <span style={{ fontFamily: FONT_UI }} className="text-[12.5px] text-[var(--ink-3)]">
              разбор открыт
            </span>
          )}
        </div>
      </div>
      {task.due && (
        <span
          style={{ fontFamily: FONT_UI }}
          className="shrink-0 rounded-full bg-[var(--warn-soft)] px-2.5 py-[3px] text-[11px] font-medium text-[var(--warn)]"
        >
          пора перерешать
        </span>
      )}
    </Link>
  )
}

export default function CodeListPage() {
  const [tasks, setTasks] = useState<CodingTaskListItem[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .getCodingTasks()
      .then((res) => setTasks(res.tasks))
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить задачи')
      })
  }, [])

  const groups = GROUP_ORDER.map((kind) => ({
    kind,
    tasks: (tasks ?? []).filter((task) => task.kind === kind),
  })).filter((group) => group.tasks.length > 0)

  return (
    <div className="mx-auto max-w-[1000px] space-y-10">
      <div>
        <h1 style={{ fontFamily: FONT_UI }} className="text-2xl font-semibold text-[var(--ink)]">
          Лайвкодинг
        </h1>
        <p style={{ fontFamily: FONT_UI }} className="mt-2 text-[15px] text-[var(--ink-2)]">
          Задачи как на собеседовании: 25 минут на попытку, лестница подсказок, разбор — только
          когда действительно застряли.
        </p>
      </div>

      {error && <p className="text-sm text-[var(--err)]">{error}</p>}

      {tasks === null ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      ) : tasks.length === 0 ? (
        <p style={{ fontFamily: FONT_UI }} className="text-sm text-[var(--ink-3)]">
          Задачи пока не добавлены.
        </p>
      ) : (
        <div className="space-y-10">
          {groups.map((group) => (
            <div key={group.kind}>
              <div
                style={{ fontFamily: FONT_CODE }}
                className="mb-2 text-[11px] font-semibold tracking-[0.08em] text-[var(--ink-3)] uppercase"
              >
                {GROUP_LABELS[group.kind]}
              </div>
              <div className="border-t border-[var(--hairline)]">
                {group.tasks.map((task) => (
                  <TaskRow key={task.slug} task={task} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
