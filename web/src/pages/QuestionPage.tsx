import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { QuestionLadderCard } from '@/components/QuestionLadderCard'
import { api, type QuestionListItem, type ReviewResult } from '@/lib/api'

const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

export default function QuestionPage() {
  const { slug = '' } = useParams<{ slug: string }>()
  const [graded, setGraded] = useState<ReviewResult | null>(null)
  const [sectionId, setSectionId] = useState<string | null>(null)
  const [sectionTitle, setSectionTitle] = useState<string | null>(null)
  const [sectionQuestions, setSectionQuestions] = useState<QuestionListItem[]>([])

  useEffect(() => {
    setGraded(null)
  }, [slug])

  useEffect(() => {
    if (!sectionId) return
    api
      .getQuestions(sectionId)
      .then((res) => setSectionQuestions(res.questions))
      .catch(() => setSectionQuestions([]))
    api
      .getSections()
      .then((res) => setSectionTitle(res.sections.find((s) => s.id === sectionId)?.title ?? null))
      .catch(() => setSectionTitle(null))
  }, [sectionId])

  const currentIndex = sectionQuestions.findIndex((q) => q.slug === slug)
  const nextSlug = currentIndex >= 0 ? sectionQuestions[currentIndex + 1]?.slug : undefined

  return (
    <div className="mx-auto max-w-[760px] pb-16">
      <div className="mb-8 flex items-center gap-4">
        <Link
          to={sectionId ? `/section/${sectionId}` : '/'}
          style={{ fontFamily: FONT_UI }}
          className="text-[13px] font-medium text-[var(--ink-2)] transition-colors hover:text-[var(--ink)]"
        >
          ← {sectionTitle ?? 'К вопросам'}
        </Link>
        {currentIndex >= 0 && sectionQuestions.length > 0 && (
          <span
            style={{ fontFamily: FONT_CODE }}
            className="ml-auto text-[11px] text-[var(--ink-3)] tabular-nums"
          >
            вопрос {currentIndex + 1} / {sectionQuestions.length}
          </span>
        )}
      </div>

      <QuestionLadderCard key={slug} slug={slug} onGraded={setGraded} onSectionResolved={setSectionId} />

      {graded && (
        <div className="mt-10 flex flex-wrap gap-3 border-t border-[var(--hairline)] pt-6">
          <Link
            to={sectionId ? `/section/${sectionId}` : '/'}
            style={{ fontFamily: FONT_UI }}
            className="rounded-full border border-[var(--accent)] px-5 py-2.5 text-[14px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)]"
          >
            К списку
          </Link>
          {nextSlug ? (
            <Link
              to={`/q/${nextSlug}`}
              style={{ fontFamily: FONT_UI }}
              className="rounded-full bg-[var(--accent)] px-5 py-2.5 text-[14px] font-medium text-[var(--bg)] transition-colors hover:bg-[var(--accent-hover)]"
            >
              Следующий вопрос
            </Link>
          ) : (
            <span
              style={{ fontFamily: FONT_UI }}
              className="rounded-full border border-[var(--hairline)] px-5 py-2.5 text-[14px] font-medium text-[var(--ink-3)]"
            >
              Это последний вопрос секции
            </span>
          )}
        </div>
      )}
    </div>
  )
}
