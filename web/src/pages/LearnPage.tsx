import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError, api, type LessonListItem, type LessonTopic } from '@/lib/api'

const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

const TOPIC_ORDER: LessonTopic[] = ['concurrency', 'go-internals', 'networks', 'os']

const TOPIC_LABELS: Record<LessonTopic, string> = {
  'go-internals': 'Память и GC',
  concurrency: 'Конкурентность',
  networks: 'Сети',
  os: 'ОС и Linux',
}

function ChapterRow({ lesson, index }: { lesson: LessonListItem; index: number }) {
  return (
    <Link
      to={`/learn/${lesson.slug}`}
      className="flex items-center gap-3 border-t border-[var(--hairline)] px-1.5 py-3 transition-colors hover:bg-[var(--accent-soft)]"
    >
      <span
        style={{ fontFamily: FONT_CODE }}
        className={cnCenter(lesson.read ? 'text-[var(--ok)]' : 'text-[var(--ink-3)]')}
      >
        {lesson.read ? '✓' : index}
      </span>
      <span
        style={{ fontFamily: FONT_UI }}
        className={
          lesson.read
            ? 'flex-1 text-[14px] text-[var(--ink-3)] line-through decoration-[#D8D3C8]'
            : 'flex-1 text-[14px] text-[var(--ink)]'
        }
      >
        {lesson.title}
        {lesson.reinforce_total > 0 && (
          <span style={{ fontFamily: FONT_CODE }} className="ml-2 text-[11px] text-[var(--ink-3)]">
            закреплено {lesson.reinforce_done}/{lesson.reinforce_total}
          </span>
        )}
      </span>
      <span style={{ fontFamily: FONT_CODE }} className="shrink-0 text-[10.5px] text-[var(--ink-3)]">
        {lesson.minutes} мин
      </span>
    </Link>
  )
}

function cnCenter(color: string): string {
  return `w-5 shrink-0 text-center text-[11px] font-semibold ${color}`
}

export default function LearnPage() {
  const [lessons, setLessons] = useState<LessonListItem[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .getLessons()
      .then((res) => setLessons(res.lessons))
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить учебник')
      })
  }, [])

  const all = lessons ?? []
  const readCount = all.filter((l) => l.read).length
  const totalCount = all.length
  const pct = totalCount > 0 ? Math.round((readCount / totalCount) * 100) : 0

  const groups = TOPIC_ORDER.map((topic) => ({
    topic,
    lessons: all.filter((l) => l.topic === topic),
  })).filter((g) => g.lessons.length > 0)

  let runningIndex = 0

  return (
    <div className="mx-auto max-w-[760px] space-y-8">
      <div>
        <h1 style={{ fontFamily: FONT_UI }} className="text-2xl font-semibold text-[var(--ink)]">
          Учебник
        </h1>
        <p style={{ fontFamily: FONT_UI }} className="mt-2 text-[15px] text-[var(--ink-2)]">
          Главы для системной подготовки: от конкурентности до внутренностей памяти.
        </p>
      </div>

      {error && <p className="text-sm text-[var(--err)]">{error}</p>}

      {lessons === null ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : totalCount === 0 ? (
        <p style={{ fontFamily: FONT_UI }} className="text-sm text-[var(--ink-3)]">
          Главы пока не добавлены.
        </p>
      ) : (
        <>
          <div className="space-y-1.5">
            <div className="flex justify-between text-[12px] text-[var(--ink-2)]" style={{ fontFamily: FONT_UI }}>
              <span>
                Прочитано {readCount} из {totalCount} глав
              </span>
              <span style={{ fontFamily: FONT_CODE }}>{pct}%</span>
            </div>
            <div className="h-1 overflow-hidden rounded-[2px] bg-[var(--hairline)]">
              <div className="h-full bg-[var(--accent)]" style={{ width: `${pct}%` }} />
            </div>
          </div>

          <div className="space-y-6">
            {groups.map((g, gi) => (
              <div key={g.topic}>
                <div
                  style={{ fontFamily: FONT_CODE }}
                  className="pb-2 text-[11px] font-semibold tracking-[0.08em] text-[var(--ink-3)] uppercase"
                >
                  Тема {gi + 1} · {TOPIC_LABELS[g.topic]}
                </div>
                <div className="border-b border-[var(--hairline)]">
                  {g.lessons.map((lesson) => {
                    runningIndex += 1
                    return <ChapterRow key={lesson.slug} lesson={lesson} index={runningIndex} />
                  })}
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
