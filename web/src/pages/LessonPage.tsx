import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Skeleton } from '@/components/ui/skeleton'
import { Markdown } from '@/components/Markdown'
import { useLessonDetail } from '@/hooks/useLessonDetail'
import {
  ApiError,
  api,
  type LessonListItem,
  type LessonRelatedItem,
  type LessonTopic,
} from '@/lib/api'

const FONT_CONTENT = 'var(--font-content, Georgia, serif)'
const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

const TOPIC_ORDER: LessonTopic[] = ['concurrency', 'go-internals', 'networks', 'os']
const TOPIC_LABELS: Record<LessonTopic, string> = {
  'go-internals': 'Память и GC',
  concurrency: 'Конкурентность',
  networks: 'Сети',
  os: 'ОС и Linux',
}

const RELATED_STATUS_LABELS: Record<string, string> = {
  new: 'новый',
  learning: 'изучается',
  review: 'на повторении',
}

function RelatedRow({ item, href }: { item: LessonRelatedItem; href: string }) {
  const pill =
    item.type === 'question'
      ? item.status === 'review'
        ? 'bg-[var(--warn-soft)] text-[var(--warn)]'
        : item.status === 'learning'
          ? 'bg-[var(--info-soft)] text-[var(--info)]'
          : 'border border-[var(--hairline)] text-[var(--ink-3)]'
      : 'bg-[var(--accent-soft)] text-[var(--ink)]'
  const label =
    item.type === 'question' ? (RELATED_STATUS_LABELS[item.status] ?? item.status) : 'задача'

  return (
    <Link
      to={href}
      className="flex items-center justify-between gap-3 border-t border-[var(--hairline)] px-1 py-3 transition-colors hover:bg-[var(--accent-soft)]"
    >
      <span style={{ fontFamily: FONT_UI }} className="min-w-0 flex-1 truncate text-[14px] text-[var(--ink)]">
        {item.title}
      </span>
      <span
        style={{ fontFamily: FONT_UI }}
        className={`shrink-0 rounded-full px-2.5 py-[3px] text-[11px] font-medium ${pill}`}
      >
        {label}
      </span>
    </Link>
  )
}

function TableOfContents({ lessons, activeSlug }: { lessons: LessonListItem[]; activeSlug: string }) {
  const groups = TOPIC_ORDER.map((topic) => ({
    topic,
    lessons: lessons.filter((l) => l.topic === topic),
  })).filter((g) => g.lessons.length > 0)

  let running = 0

  return (
    <nav className="flex flex-col gap-4">
      <Link
        to="/learn"
        style={{ fontFamily: FONT_UI }}
        className="text-[13px] font-medium text-[var(--ink-2)] transition-colors hover:text-[var(--ink)]"
      >
        ← Оглавление
      </Link>
      {groups.map((g, gi) => (
        <div key={g.topic}>
          <div
            style={{ fontFamily: FONT_CODE }}
            className="pb-1 text-[11px] font-semibold tracking-[0.08em] text-[var(--ink-3)] uppercase"
          >
            Тема {gi + 1} · {TOPIC_LABELS[g.topic]}
          </div>
          {g.lessons.map((lesson) => {
            running += 1
            const active = lesson.slug === activeSlug
            return (
              <Link
                key={lesson.slug}
                to={`/learn/${lesson.slug}`}
                className={
                  active
                    ? 'flex items-center gap-2.5 rounded-[8px] bg-[var(--accent-soft)] px-2 py-2.5'
                    : 'flex items-center gap-2.5 border-t border-[var(--hairline)] px-2 py-2.5 transition-colors hover:bg-[var(--accent-soft)]'
                }
              >
                <span
                  style={{ fontFamily: FONT_CODE }}
                  className={`w-4 shrink-0 text-center text-[11px] font-semibold ${
                    lesson.read ? 'text-[var(--ok)]' : active ? 'text-[var(--ink)]' : 'text-[var(--ink-3)]'
                  }`}
                >
                  {lesson.read ? '✓' : active ? '▸' : running}
                </span>
                <span
                  style={{ fontFamily: FONT_UI }}
                  className={`flex-1 text-[14px] ${
                    lesson.read
                      ? 'text-[var(--ink-3)] line-through decoration-[#D8D3C8]'
                      : active
                        ? 'font-medium text-[var(--ink)]'
                        : 'text-[var(--ink)]'
                  }`}
                >
                  {lesson.title}
                </span>
                <span style={{ fontFamily: FONT_CODE }} className="shrink-0 text-[10.5px] text-[var(--ink-3)]">
                  {lesson.minutes} мин
                </span>
              </Link>
            )
          })}
        </div>
      ))}
    </nav>
  )
}

export default function LessonPage() {
  const { slug = '' } = useParams<{ slug: string }>()
  const { data: lesson, loading, error, setData } = useLessonDetail(slug)
  const [lessons, setLessons] = useState<LessonListItem[]>([])
  const [marking, setMarking] = useState(false)
  const [progress, setProgress] = useState(0)
  const articleRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api
      .getLessons()
      .then((res) => setLessons(res.lessons))
      .catch(() => setLessons([]))
  }, [])

  // Прогресс чтения по скроллу (для тонкой полосы на мобайле).
  useEffect(() => {
    function onScroll() {
      const el = articleRef.current
      if (!el) return
      const rect = el.getBoundingClientRect()
      const scrollable = el.offsetHeight - window.innerHeight
      const scrolled = -rect.top
      const p =
        scrollable > 0
          ? Math.min(1, Math.max(0, scrolled / scrollable))
          : rect.bottom <= window.innerHeight
            ? 1
            : 0
      setProgress(p)
    }
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll)
    onScroll()
    return () => {
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [lesson])

  async function handleMarkRead() {
    if (!lesson) return
    setMarking(true)
    try {
      await api.markLessonRead(lesson.slug)
      setData({ ...lesson, read: true })
      setLessons((prev) => prev.map((l) => (l.slug === lesson.slug ? { ...l, read: true } : l)))
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Не удалось отметить главу прочитанной')
    } finally {
      setMarking(false)
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-[620px] space-y-6 pb-16">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !lesson) {
    return <p className="text-sm text-[var(--err)]">{error ?? 'Глава не найдена'}</p>
  }

  const remaining = Math.max(0, Math.round(lesson.minutes * (1 - progress)))

  const readButton = lesson.read ? (
    <span
      style={{ fontFamily: FONT_UI }}
      className="inline-flex items-center gap-1.5 rounded-full border border-[var(--ok)] px-6 py-3 text-[14px] font-medium text-[var(--ok)]"
    >
      ✓ Прочитано
    </span>
  ) : (
    <button
      type="button"
      onClick={handleMarkRead}
      disabled={marking}
      style={{ fontFamily: FONT_UI }}
      className="rounded-full bg-[var(--accent)] px-7 py-3 text-[14px] font-medium text-[var(--bg)] transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-60"
    >
      {marking ? 'Отмечаем…' : '✓ Прочитано'}
    </button>
  )

  return (
    <>
      {/* Тонкий прогресс чтения — мобайл (1m) */}
      <div className="sticky top-0 z-20 mb-4 h-[3px] bg-[var(--hairline)] sm:hidden">
        <div
          className="h-full bg-[var(--accent)] transition-[width] duration-150"
          style={{ width: `${Math.round(progress * 100)}%` }}
        />
      </div>

      <div className="mx-auto grid max-w-[1000px] gap-x-12 lg:grid-cols-[260px_minmax(0,600px)] lg:justify-center">
        {/* Оглавление — десктоп */}
        <aside className="hidden lg:block">
          <div className="sticky top-8">
            {lessons.length > 0 && <TableOfContents lessons={lessons} activeSlug={lesson.slug} />}
          </div>
        </aside>

        {/* Колонка чтения */}
        <article ref={articleRef} className="min-w-0 pb-24 lg:pb-16">
          <div className="mb-6 flex items-center justify-between lg:hidden">
            <Link
              to="/learn"
              style={{ fontFamily: FONT_UI }}
              className="text-[13px] font-medium text-[var(--ink-2)] transition-colors hover:text-[var(--ink)]"
            >
              ← Оглавление
            </Link>
            <span style={{ fontFamily: FONT_CODE }} className="text-[10.5px] text-[var(--ink-3)]">
              {lesson.read ? 'прочитано' : `осталось ~${remaining} мин`}
            </span>
          </div>

          <div style={{ fontFamily: FONT_CODE }} className="mb-4 text-[11px] tracking-[0.06em] text-[var(--ink-3)] uppercase">
            {lesson.minutes} мин чтения
            {lesson.tags.length > 0 && <> · {lesson.tags.map((t) => `#${t}`).join(' ')}</>}
          </div>

          <h1
            style={{ fontFamily: FONT_CONTENT }}
            className="mb-6 text-[30px] leading-[1.3] font-bold text-[var(--ink)] [text-wrap:pretty] sm:text-[32px]"
          >
            {lesson.title}
          </h1>

          <Markdown size="lg">{lesson.body_md}</Markdown>

          <div className="mt-8 hidden justify-center border-t border-[var(--hairline)] pt-8 sm:flex">
            {readButton}
          </div>

          {lesson.read && lesson.related.length > 0 && (
            <div className="mt-8 rounded-[var(--r-md)] border border-[var(--hairline)] bg-[var(--surface)] px-6 py-5">
              <p style={{ fontFamily: FONT_UI }} className="mb-3 text-[15px] font-semibold text-[var(--ink)]">
                Закрепите материал
              </p>
              <div>
                {lesson.related.map((r) => (
                  <RelatedRow
                    key={`${r.type}-${r.slug}`}
                    item={r}
                    href={r.type === 'question' ? `/q/${r.slug}` : `/code/${r.slug}`}
                  />
                ))}
              </div>
            </div>
          )}
        </article>
      </div>

      {/* Sticky «✓ Прочитано» — мобайл */}
      {!lesson.read && (
        <div className="sticky bottom-0 z-20 border-t border-[var(--hairline)] bg-[var(--bg)] py-3 sm:hidden">
          <button
            type="button"
            onClick={handleMarkRead}
            disabled={marking}
            style={{ fontFamily: FONT_UI }}
            className="w-full rounded-full bg-[var(--accent)] px-5 py-3.5 text-[15px] font-medium text-[var(--bg)] transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-60"
          >
            {marking ? 'Отмечаем…' : '✓ Прочитано · к вопросам'}
          </button>
        </div>
      )}
    </>
  )
}
