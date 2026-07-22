import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { PencilIcon, XIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { SegmentedProgress } from '@/components/ui/progress'
import { SectionCard } from '@/components/SectionCard'
import { EmptyState } from '@/components/EmptyState'
import { DifficultyBadge } from '@/components/DifficultyBadge'
import {
  api,
  type CodingTaskListItem,
  type LessonListItem,
  type Me,
  type Section,
  type Stats,
} from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { daysUntil, greeting, plural } from '@/lib/format'

// ---- Инлайн-редактор профиля (имя + дата собеседования) ----
function ProfileEditor({
  me,
  onClose,
  onSaved,
}: {
  me: Me | null
  onClose: () => void
  onSaved: (me: Me) => void
}) {
  const [name, setName] = useState(me?.name ?? '')
  const [date, setDate] = useState(me?.interview_date ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      const updated = await api.updateMe({
        name: name.trim() || null,
        interview_date: date || null,
      })
      onSaved(updated)
      onClose()
    } catch {
      setError('Не удалось сохранить. Попробуйте ещё раз.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-40 flex items-center justify-center px-5"
      style={{ backgroundColor: 'var(--scrim)' }}
      onClick={onClose}
    >
      <div
        className="w-full max-w-[360px] rounded-[10px] border border-hairline bg-surface p-6 shadow-[var(--shadow-md)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-[17px] font-semibold text-ink">Профиль</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Закрыть"
            className="inline-flex size-8 items-center justify-center rounded-full text-ink-3 transition-colors hover:bg-accent-soft hover:text-ink"
          >
            <XIcon className="size-4" />
          </button>
        </div>
        <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="profile-name" className="text-[12.5px] text-ink-2">
              Имя
            </Label>
            <Input
              id="profile-name"
              value={name}
              placeholder="Как к вам обращаться"
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="profile-date" className="text-[12.5px] text-ink-2">
              Дата собеседования
            </Label>
            <Input
              id="profile-date"
              type="date"
              value={date ?? ''}
              onChange={(e) => setDate(e.target.value)}
            />
          </div>
          {error && <p className="text-[12.5px] text-err">{error}</p>}
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="outline" size="sm" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" size="sm" disabled={saving}>
              {saving ? 'Сохраняем…' : 'Сохранить'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ---- Счётчик через hairline (десктоп) ----
function Counter({ value, suffix, label }: { value: number; suffix?: string; label: string }) {
  return (
    <div className="flex flex-col gap-1 bg-bg px-6 py-[18px]">
      <span className="font-mono text-[26px] font-semibold leading-none text-ink">
        {value}
        {suffix && <span className="text-[15px] text-ink-3"> {suffix}</span>}
      </span>
      <span className="text-[12.5px] text-ink-2">{label}</span>
    </div>
  )
}

export default function DashboardPage() {
  const { user } = useAuth()
  const location = useLocation()
  const [me, setMe] = useState<Me | null>(null)
  const [sections, setSections] = useState<Section[] | null>(null)
  const [stats, setStats] = useState<Stats | null>(null)
  const [lessons, setLessons] = useState<LessonListItem[] | null>(null)
  const [coding, setCoding] = useState<CodingTaskListItem[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const sectionsRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    Promise.allSettled([
      api.getMe(),
      api.getSections(),
      api.getStats(),
      api.getLessons(),
      api.getCodingTasks(),
    ]).then(([meR, secR, statR, lesR, codR]) => {
      if (cancelled) return
      if (meR.status === 'fulfilled') setMe(meR.value)
      if (secR.status === 'fulfilled') setSections(secR.value.sections)
      else setError('Не удалось загрузить секции')
      if (statR.status === 'fulfilled') setStats(statR.value)
      if (lesR.status === 'fulfilled') setLessons(lesR.value.lessons)
      if (codR.status === 'fulfilled') setCoding(codR.value.tasks)
    })
    return () => {
      cancelled = true
    }
  }, [])

  // Переход по пункту «Вопросы» (/#sections) — прокрутка к списку секций.
  useEffect(() => {
    if (location.hash === '#sections' && sectionsRef.current) {
      sectionsRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }, [location.hash, sections])

  const displayName = me?.name?.trim() || user?.email?.split('@')[0] || ''
  const dueReview = stats?.due_today ?? 0
  const codingDue = stats?.coding?.due ?? 0
  const lessonsRead = stats?.lessons?.read ?? lessons?.filter((l) => l.read).length ?? 0
  const lessonsTotal = stats?.lessons?.total ?? lessons?.length ?? 0

  const days = daysUntil(me?.interview_date)
  const currentChapter = lessons?.find((l) => !l.read) ?? null
  const dueTasks = (coding ?? []).filter((t) => t.due).slice(0, 2)
  const newTask = (coding ?? []).find((t) => t.status === 'new') ?? null

  // Подзаголовок: дедлайн + план на сегодня.
  const planParts: string[] = []
  if (dueReview > 0) planParts.push('повторение')
  if (currentChapter) planParts.push('одна глава')
  const deadlineText =
    days == null
      ? null
      : days > 0
        ? `До собеседования — ${days} ${plural(days, ['день', 'дня', 'дней'])}.`
        : days === 0
          ? 'Собеседование сегодня.'
          : null
  const planText = planParts.length ? `Сегодня по плану: ${planParts.join(' и ')}.` : null
  const subline = [deadlineText, planText].filter(Boolean).join(' ')

  const estimate = Math.max(5, dueReview * 2)

  return (
    <div className="mx-auto flex max-w-[860px] flex-col gap-9">
      {editing && (
        <ProfileEditor me={me} onClose={() => setEditing(false)} onSaved={setMe} />
      )}

      {/* Приветствие + CTA */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="flex min-w-0 flex-col gap-1.5">
          <div className="flex items-center gap-2">
            <h1 className="text-[24px] font-semibold text-ink">
              {greeting()}
              {displayName && `, ${displayName}`}
            </h1>
            <button
              type="button"
              onClick={() => setEditing(true)}
              aria-label="Изменить профиль"
              title="Изменить имя и дату собеседования"
              className="inline-flex size-7 items-center justify-center rounded-full text-ink-3 transition-colors hover:bg-accent-soft hover:text-ink"
            >
              <PencilIcon className="size-[15px]" />
            </button>
          </div>
          {subline ? (
            <p className="text-sm text-ink-2">{subline}</p>
          ) : (
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="self-start text-sm text-ink-3 underline-offset-4 hover:text-ink hover:underline"
            >
              Укажите дату собеседования
            </button>
          )}
        </div>
        {dueReview > 0 && (
          <Button asChild className="w-full sm:w-auto">
            <Link to="/review">
              <span>Повторить {dueReview} {plural(dueReview, ['вопрос', 'вопроса', 'вопросов'])}</span>
              <span className="font-mono text-xs opacity-80 sm:hidden">~{estimate} мин</span>
            </Link>
          </Button>
        )}
      </div>

      {/* Счётчики: десктоп — 3 через hairline; мобайл — 2 карточки */}
      <div className="hidden grid-cols-3 gap-px overflow-hidden border-y border-hairline bg-hairline sm:grid">
        <Counter value={dueReview} label="вопросов к повторению сегодня" />
        <Counter value={codingDue} label="задачи перерешать по памяти" />
        <Counter value={lessonsRead} suffix={`/ ${lessonsTotal}`} label="глав учебника прочитано" />
      </div>
      <div className="grid grid-cols-2 gap-2.5 sm:hidden">
        <div className="flex flex-col gap-0.5 rounded-[10px] border border-hairline bg-surface px-3.5 py-3">
          <span className="font-mono text-[22px] font-semibold leading-none text-ink">{codingDue}</span>
          <span className="text-[11.5px] text-ink-2">перерешать задачи</span>
        </div>
        <div className="flex flex-col gap-0.5 rounded-[10px] border border-hairline bg-surface px-3.5 py-3">
          <span className="font-mono text-[22px] font-semibold leading-none text-ink">
            {lessonsRead}
            <span className="text-[13px] text-ink-3">/{lessonsTotal}</span>
          </span>
          <span className="text-[11.5px] text-ink-2">глав прочитано</span>
        </div>
      </div>

      {/* Секции вопросов */}
      <div ref={sectionsRef} id="sections" className="scroll-mt-20">
        <h2 className="mb-2.5 text-[17px] font-semibold text-ink">Секции вопросов</h2>
        {error && !sections && <p className="text-sm text-err">{error}</p>}
        {sections === null && !error ? (
          <div className="flex flex-col gap-px">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : sections && sections.length === 0 ? (
          <EmptyState
            title="Секции пока не добавлены"
            description="Как только появятся вопросы, они возникнут здесь по разделам."
          />
        ) : (
          <div className="border-b border-hairline">
            {sections?.map((s) => (
              <SectionCard key={s.id} section={s} />
            ))}
          </div>
        )}
      </div>

      {/* Учебник + Лайвкодинг */}
      {(lessons !== null || coding !== null) && (
        <div className="grid gap-6 sm:grid-cols-2">
          {lessons !== null && (
            <div className="flex flex-col gap-3">
              <h2 className="text-[17px] font-semibold text-ink">Учебник</h2>
              <div className="flex flex-col gap-3 rounded-[10px] border border-hairline bg-surface p-5">
                {currentChapter ? (
                  <>
                    <span className="text-xs text-ink-3">Вы остановились на</span>
                    <span className="font-serif text-[17px] font-bold text-ink">
                      {currentChapter.title}
                    </span>
                    <div className="flex items-center gap-3">
                      <SegmentedProgress
                        total={lessonsTotal || 1}
                        filled={lessonsRead}
                        className="flex-1"
                      />
                      <span className="shrink-0 font-mono text-[10.5px] text-ink-3">
                        {lessonsRead}/{lessonsTotal} · ~{currentChapter.minutes} мин
                      </span>
                    </div>
                    <Button asChild variant="outline" size="sm" className="self-start">
                      <Link to={`/learn/${currentChapter.slug}`}>Продолжить чтение</Link>
                    </Button>
                  </>
                ) : (
                  <>
                    <span className="font-serif text-[17px] font-bold text-ink">
                      Все главы прочитаны
                    </span>
                    <Button asChild variant="outline" size="sm" className="self-start">
                      <Link to="/learn">К учебнику</Link>
                    </Button>
                  </>
                )}
              </div>
            </div>
          )}

          {coding !== null && (dueTasks.length > 0 || newTask) && (
            <div className="flex flex-col gap-3">
              <h2 className="text-[17px] font-semibold text-ink">Лайвкодинг</h2>
              <div className="flex flex-col rounded-[10px] border border-hairline bg-surface p-5">
                {dueTasks.map((t, i) => (
                  <Link
                    key={t.slug}
                    to={`/code/${t.slug}`}
                    className={
                      'flex items-center justify-between gap-2 py-2.5 first:pt-0 ' +
                      (i > 0 || newTask ? 'border-t border-hairline' : '')
                    }
                  >
                    <span className="min-w-0 truncate text-sm font-medium text-ink">{t.title}</span>
                    <Badge variant="warn">перерешать</Badge>
                  </Link>
                ))}
                {newTask && (
                  <Link
                    to={`/code/${newTask.slug}`}
                    className={
                      'flex items-center justify-between gap-2 py-2.5 first:pt-0 ' +
                      (dueTasks.length > 0 ? 'border-t border-hairline' : '')
                    }
                  >
                    <span className="min-w-0 truncate text-sm font-medium text-ink-2">
                      Новая: {newTask.title}
                    </span>
                    <DifficultyBadge difficulty={newTask.difficulty} />
                  </Link>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
