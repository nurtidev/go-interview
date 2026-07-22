import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Skeleton } from '@/components/ui/skeleton'
import { DifficultyBadge } from '@/components/DifficultyBadge'
import { StatusBadge } from '@/components/StatusBadge'
import { ApiError, api, type QuestionListItem, type Section } from '@/lib/api'

const FONT_CONTENT = 'var(--font-content, Georgia, serif)'
const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'

export default function SectionPage() {
  const { id = '' } = useParams<{ id: string }>()
  const [section, setSection] = useState<Section | null>(null)
  const [questions, setQuestions] = useState<QuestionListItem[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setQuestions(null)
    setSection(null)
    setError(null)
    Promise.all([api.getSections(), api.getQuestions(id)])
      .then(([sectionsRes, questionsRes]) => {
        setSection(sectionsRes.sections.find((s) => s.id === id) ?? null)
        setQuestions(questionsRes.questions)
      })
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить вопросы секции')
      })
  }, [id])

  return (
    <div className="mx-auto max-w-[1000px] space-y-8">
      <div>
        <h1 style={{ fontFamily: FONT_CONTENT }} className="text-3xl font-bold text-[var(--ink)]">
          {section?.title ?? 'Секция'}
        </h1>
        {section?.description && (
          <p style={{ fontFamily: FONT_UI }} className="mt-2 text-[15px] text-[var(--ink-2)]">
            {section.description}
          </p>
        )}
      </div>

      {error && <p className="text-sm text-[var(--err)]">{error}</p>}

      {questions === null ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : questions.length === 0 ? (
        <p style={{ fontFamily: FONT_UI }} className="text-sm text-[var(--ink-3)]">
          В этой секции пока нет вопросов.
        </p>
      ) : (
        <div className="border-t border-[var(--hairline)]">
          {questions.map((q) => (
            <Link
              key={q.slug}
              to={`/q/${q.slug}`}
              className="flex flex-col gap-2 border-b border-[var(--hairline)] px-1.5 py-5 transition-colors hover:bg-[var(--accent-soft)] sm:flex-row sm:items-center sm:justify-between"
            >
              <p
                style={{ fontFamily: FONT_CONTENT }}
                className="text-lg font-bold text-[var(--ink)]"
              >
                {q.title}
              </p>
              <div className="flex shrink-0 flex-wrap items-center gap-2 sm:justify-end">
                <DifficultyBadge difficulty={q.difficulty} />
                {q.tags.length > 0 && (
                  <span style={{ fontFamily: FONT_UI }} className="text-[13px] text-[var(--ink-3)]">
                    {q.tags.map((t) => `#${t}`).join(' ')}
                  </span>
                )}
                <StatusBadge status={q.status} />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
