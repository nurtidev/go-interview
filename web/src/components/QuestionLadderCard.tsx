import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Skeleton } from '@/components/ui/skeleton'
import { Markdown } from '@/components/Markdown'
import { DifficultyBadge } from '@/components/DifficultyBadge'
import { useQuestionDetail } from '@/hooks/useQuestionDetail'
import { ApiError, api, type AnswerLevelName, type Grade, type ReviewResult } from '@/lib/api'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'

const FONT_CONTENT = 'var(--font-content, Georgia, serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'
const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'

const LEVEL_ORDER: AnswerLevelName[] = ['middle', 'senior', 'deep']

const LEVEL_META: Record<AnswerLevelName, { subtitle: string; tone: string }> = {
  middle: { subtitle: 'база', tone: 'text-[var(--ink-2)]' },
  senior: { subtitle: 'нюансы', tone: 'text-[var(--ink)]' },
  deep: { subtitle: 'внутренности', tone: 'text-[var(--staff)]' },
}

// Самооценка SM-2 (спека 1b): фиксированный порядок, семантический цвет на -soft
// тинте, вторая строка mono с интервалом следующего показа.
const GRADES: Array<{ grade: Grade; label: string; interval: string; color: string; soft: string }> = [
  { grade: 'again', label: 'Снова', interval: '10 мин', color: 'var(--err)', soft: 'var(--err-soft)' },
  { grade: 'hard', label: 'Трудно', interval: '1 день', color: 'var(--warn)', soft: 'var(--warn-soft)' },
  { grade: 'good', label: 'Хорошо', interval: '3 дня', color: 'var(--ok)', soft: 'var(--ok-soft)' },
  { grade: 'easy', label: 'Легко', interval: '7 дней', color: 'var(--info)', soft: 'var(--info-soft)' },
]

interface Props {
  slug: string
  mode?: 'study' | 'review'
  onGraded?: (result: ReviewResult) => void
  onSectionResolved?: (sectionId: string) => void
}

export function QuestionLadderCard({ slug, mode = 'study', onGraded, onSectionResolved }: Props) {
  const { data, loading, error } = useQuestionDetail(slug)
  const [revealedCount, setRevealedCount] = useState(0)
  const [submitting, setSubmitting] = useState<Grade | null>(null)
  const [selected, setSelected] = useState<Grade | null>(null)
  const [gradeResult, setGradeResult] = useState<ReviewResult | null>(null)

  useEffect(() => {
    if (data) onSectionResolved?.(data.section)
  }, [data, onSectionResolved])

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-10 w-56" />
      </div>
    )
  }

  if (error || !data) {
    return <p className="text-sm text-[var(--err)]">{error ?? 'Вопрос не найден'}</p>
  }

  const question = data
  const levels = LEVEL_ORDER.filter((lvl) => question.answer_levels.some((a) => a.level === lvl))
  const allRevealed = revealedCount >= levels.length
  const showRail = mode === 'study'
  const nextLevel = levels[revealedCount]

  async function handleGrade(grade: Grade) {
    setSelected(grade)
    setSubmitting(grade)
    try {
      const result = await api.submitReview(question.slug, grade)
      setGradeResult(result)
      toast.success(`Следующее повторение: ${formatDate(result.due_at)}`)
      onGraded?.(result)
    } catch (e) {
      setSelected(null)
      toast.error(e instanceof ApiError ? e.message : 'Не удалось сохранить оценку')
    } finally {
      setSubmitting(null)
    }
  }

  function revealAll() {
    setRevealedCount(levels.length)
  }

  // ---------- Режим повторения: до раскрытия — минимальный экран ----------
  if (mode === 'review' && revealedCount === 0) {
    return (
      <div className="flex flex-col items-center gap-6 text-center">
        <div className="flex flex-wrap items-center justify-center gap-2">
          <DifficultyBadge difficulty={question.difficulty} />
        </div>
        <h1
          style={{ fontFamily: FONT_CONTENT }}
          className="text-[25px] leading-[1.4] font-bold text-[var(--ink)] [text-wrap:pretty]"
        >
          {question.title}
        </h1>
        {question.question_md && (
          <div className="max-w-[520px]">
            <Markdown size="md">{question.question_md}</Markdown>
          </div>
        )}
        <button
          type="button"
          onClick={revealAll}
          style={{ fontFamily: FONT_UI }}
          className="rounded-full border border-[var(--accent)] px-6 py-2.5 text-[14px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)]"
        >
          Показать ответ
        </button>
        <p style={{ fontFamily: FONT_UI }} className="text-[12px] text-[var(--ink-3)]">
          Ответьте вслух, затем сверьтесь. Лестница откроется целиком.
        </p>
      </div>
    )
  }

  const centered = mode === 'review'

  return (
    <div className="space-y-8">
      {/* ---------- Шапка ---------- */}
      <div className={cn('space-y-3', centered && 'text-center')}>
        <div className={cn('flex flex-wrap items-center gap-2', centered && 'justify-center')}>
          <DifficultyBadge difficulty={question.difficulty} />
          {question.status === 'review' && (
            <span
              style={{ fontFamily: FONT_UI }}
              className="inline-flex items-center rounded-full bg-[var(--warn-soft)] px-2.5 py-[3px] text-[11.5px] font-medium text-[var(--warn)]"
            >
              к повторению
            </span>
          )}
          {question.tags.length > 0 && !centered && (
            <span style={{ fontFamily: FONT_UI }} className="text-[13px] text-[var(--ink-3)]">
              {question.tags.map((t) => `#${t}`).join(' ')}
            </span>
          )}
        </div>
        <h1
          style={{ fontFamily: FONT_CONTENT }}
          className={cn(
            'font-bold text-[var(--ink)] [text-wrap:pretty]',
            centered ? 'text-[25px] leading-[1.4]' : 'text-[26px] leading-[1.35]',
          )}
        >
          {question.title}
        </h1>
        {!centered && (
          <>
            {question.question_md && <Markdown size="md">{question.question_md}</Markdown>}
            <p style={{ fontFamily: FONT_UI }} className="text-[15px] leading-[1.6] text-[var(--ink-2)]">
              Сначала ответьте про себя вслух — как на собеседовании. Затем раскрывайте уровни по
              одному.
            </p>
          </>
        )}
      </div>

      {/* ---------- Лестница уровней ---------- */}
      <div className={cn('space-y-0 border-t border-[var(--hairline)] pt-8', centered && 'text-left')}>
        {levels.map((lvl, idx) => {
          const entry = question.answer_levels.find((a) => a.level === lvl)
          if (!entry) return null
          const revealed = idx < revealedCount
          const isNext = idx === revealedCount && mode === 'study'
          const isFuture = idx > revealedCount
          const meta = LEVEL_META[lvl]

          return (
            <div
              key={lvl}
              className={cn('gap-4 sm:gap-5', isFuture ? 'hidden sm:flex' : 'flex')}
            >
              {/* Рельса-стёпер 1-2-3 (десктоп, режим изучения) */}
              {showRail && (
                <div className="hidden w-7 shrink-0 flex-col items-center sm:flex">
                  <span
                    style={{ fontFamily: FONT_CODE }}
                    className={cn(
                      'flex size-7 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold',
                      revealed
                        ? 'bg-[var(--accent)] text-[var(--bg)]'
                        : isNext
                          ? 'border-2 border-[var(--accent)] text-[var(--ink)]'
                          : 'border border-[var(--hairline)] text-[var(--ink-3)]',
                    )}
                  >
                    {idx + 1}
                  </span>
                  {idx < levels.length - 1 && (
                    <span
                      className="w-[1.5px] flex-1"
                      style={{ background: revealed ? 'var(--accent)' : 'var(--hairline)' }}
                    />
                  )}
                </div>
              )}

              <div className="min-w-0 flex-1 pb-7">
                <div
                  style={{ fontFamily: FONT_UI }}
                  className={cn(
                    'mb-2 text-[12px] font-semibold uppercase tracking-[0.05em]',
                    meta.tone,
                  )}
                >
                  {lvl} — {meta.subtitle}
                </div>

                {revealed && <Markdown size="md">{entry.text_md}</Markdown>}

                {isNext && (
                  <>
                    {/* Десктоп — инлайн-кнопка раскрытия */}
                    <button
                      type="button"
                      onClick={() => setRevealedCount((c) => c + 1)}
                      style={{ fontFamily: FONT_UI }}
                      className="hidden rounded-full border border-[var(--accent)] px-[18px] py-2 text-[14px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)] sm:inline-block"
                    >
                      Раскрыть уровень {lvl}
                    </button>
                    {/* Мобайл — заглушка с fade (кнопка раскрытия закреплена снизу) */}
                    <div
                      className="h-11 rounded-[6px] sm:hidden"
                      style={{
                        background: 'linear-gradient(var(--accent-soft), transparent)',
                      }}
                    />
                  </>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* ---------- Самооценка ---------- */}
      {(mode === 'review' || revealedCount > 0) && (
        <div className={cn('border-t border-[var(--hairline)] pt-6', centered && 'text-center')}>
          {gradeResult ? (
            <p style={{ fontFamily: FONT_UI }} className="text-[14px] text-[var(--ink-2)]">
              Оценка сохранена. Следующее повторение: {formatDate(gradeResult.due_at)}
            </p>
          ) : (
            <div className={cn('space-y-3', centered && 'flex flex-col items-center')}>
              <p style={{ fontFamily: FONT_UI }} className="text-[13.5px] font-medium text-[var(--ink)]">
                Насколько хорошо вы ответили?
              </p>
              <div
                className={cn(
                  'flex flex-wrap gap-2 transition-opacity',
                  centered && 'justify-center',
                  allRevealed ? 'opacity-100' : 'opacity-55',
                )}
              >
                {GRADES.map((g) => (
                  <button
                    key={g.grade}
                    type="button"
                    disabled={!allRevealed || submitting !== null}
                    onClick={() => handleGrade(g.grade)}
                    style={{ color: g.color, background: g.soft, fontFamily: FONT_UI }}
                    className={cn(
                      'flex flex-col items-center gap-px rounded-full border-2 px-4 py-1.5 text-[13px] font-medium transition',
                      selected === g.grade ? 'border-current' : 'border-transparent',
                      allRevealed && submitting === null
                        ? 'cursor-pointer hover:brightness-[0.97]'
                        : 'cursor-default',
                    )}
                  >
                    <span>{submitting === g.grade ? 'Сохраняем…' : g.label}</span>
                    <span style={{ fontFamily: FONT_CODE }} className="text-[10.5px] opacity-65">
                      {g.interval}
                    </span>
                  </button>
                ))}
              </div>
              {!allRevealed && (
                <p style={{ fontFamily: FONT_UI }} className="text-[11.5px] text-[var(--ink-3)]">
                  Активируется после раскрытия всех уровней
                </p>
              )}
            </div>
          )}
        </div>
      )}

      {/* ---------- Вопросы вдогонку (только в изучении, после раскрытия всех) ---------- */}
      {mode === 'study' && allRevealed && question.follow_ups.length > 0 && (
        <div className="space-y-3 border-t border-[var(--hairline)] pt-6">
          <p style={{ fontFamily: FONT_UI }} className="text-[13.5px] font-medium text-[var(--ink)]">
            Вопросы вдогонку, которые может задать интервьюер
          </p>
          <ul
            style={{ fontFamily: FONT_CONTENT }}
            className="ml-6 list-disc space-y-2 text-[17px] leading-[1.65] text-[var(--ink)]"
          >
            {question.follow_ups.map((f) => (
              <li key={f}>{f}</li>
            ))}
          </ul>
        </div>
      )}

      {/* ---------- Мобайл: кнопка раскрытия закреплена снизу ---------- */}
      {mode === 'study' && !allRevealed && nextLevel && (
        <div className="sticky bottom-0 z-10 border-t border-[var(--hairline)] bg-[var(--bg)] py-3 sm:hidden">
          <button
            type="button"
            onClick={() => setRevealedCount((c) => c + 1)}
            style={{ fontFamily: FONT_UI }}
            className="w-full rounded-full bg-[var(--accent)] px-5 py-3.5 text-[15px] font-medium text-[var(--bg)] transition-colors hover:bg-[var(--accent-hover)]"
          >
            Раскрыть уровень {nextLevel}
          </button>
        </div>
      )}
    </div>
  )
}
