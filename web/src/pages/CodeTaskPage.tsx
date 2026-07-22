import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Loader2Icon, PlayIcon } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { Markdown } from '@/components/Markdown'
import { DifficultyBadge } from '@/components/DifficultyBadge'
import { CodeEditor } from '@/components/coding/CodeEditor'
import { DataTable } from '@/components/coding/DataTable'
import { GoConsole, GoRunSummary } from '@/components/coding/GoConsole'
import { SqlResultView, type SqlOutcome } from '@/components/coding/SqlResultView'
import { useAttemptTimer, formatHintCountdown } from '@/hooks/useAttemptTimer'
import { useCodingTaskDetail } from '@/hooks/useCodingTaskDetail'
import { ApiError, api, type CodingTaskDetail, type RunResponse } from '@/lib/api'
import {
  compareRows,
  countTestResults,
  readDraft,
  readOpenedHints,
  writeDraft,
  writeOpenedHints,
} from '@/lib/coding'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'

const FONT_CONTENT = 'var(--font-content, Georgia, serif)'
const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

type WorkspaceTab = 'condition' | 'code' | 'result'

function panelClasses(active: WorkspaceTab, name: WorkspaceTab): string {
  return cn('[@media(min-width:900px)]:block', active === name ? 'block' : 'hidden')
}

function SectionHead({ children }: { children: string }) {
  return (
    <div
      style={{ fontFamily: FONT_CODE }}
      className="mb-3.5 text-[11px] font-semibold tracking-[0.08em] whitespace-nowrap text-[var(--ink-3)] uppercase"
    >
      {children}
    </div>
  )
}

export default function CodeTaskPage() {
  const { slug = '' } = useParams<{ slug: string }>()
  const { data: task, loading, error, setData } = useCodingTaskDetail(slug)

  if (loading) {
    return (
      <div className="mx-auto max-w-[1200px] space-y-6">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-80 w-full" />
      </div>
    )
  }

  if (error || !task) {
    return <p className="text-sm text-[var(--err)]">{error ?? 'Задача не найдена'}</p>
  }

  return <TaskWorkspace key={slug} slug={slug} task={task} setTask={setData} />
}

interface WorkspaceProps {
  slug: string
  task: CodingTaskDetail
  setTask: (task: CodingTaskDetail) => void
}

function TaskWorkspace({ slug, task, setTask }: WorkspaceProps) {
  const hintsCount = task.hints.length

  const [code, setCode] = useState<string>(
    () => readDraft(slug) ?? task.last_code ?? task.starter_code,
  )
  const [openedHints, setOpenedHints] = useState<number>(() => readOpenedHints(slug, hintsCount))
  const [activeTab, setActiveTab] = useState<WorkspaceTab>('condition')
  const [running, setRunning] = useState(false)
  const [goResult, setGoResult] = useState<RunResponse | null>(null)
  const [sqlOutcome, setSqlOutcome] = useState<SqlOutcome | null>(null)
  const [solutionMd, setSolutionMd] = useState<string | null>(null)
  const [solutionLoading, setSolutionLoading] = useState(false)
  const [conditionCollapsed, setConditionCollapsed] = useState(false)

  const timer = useAttemptTimer(slug, task.difficulty, openedHints, hintsCount)

  // Автосохранение черновика (debounce 1с).
  useEffect(() => {
    const id = window.setTimeout(() => writeDraft(slug, code), 1000)
    return () => window.clearTimeout(id)
  }, [slug, code])

  useEffect(() => {
    writeOpenedHints(slug, openedHints)
  }, [slug, openedHints])

  const solved = task.status === 'solved'
  const canGiveUp = !solved && !task.gave_up
  const allHintsOpen = openedHints >= hintsCount
  const dirty = code !== (task.last_code ?? task.starter_code)

  const runLabel = task.kind === 'go' ? 'Запустить тесты' : 'Выполнить запрос'
  const busyLabel = task.kind === 'go' ? 'Запускаем…' : 'Выполняем…'
  const fileName = task.kind === 'go' ? 'solution.go' : 'query.sql'

  const failedCount =
    task.kind === 'go'
      ? goResult
        ? countTestResults(goResult.output).failed
        : 0
      : sqlOutcome
        ? sqlOutcome.error || (sqlOutcome.diff && !sqlOutcome.diff.matched)
          ? 1
          : 0
        : 0

  const tabs: Array<{ id: WorkspaceTab; label: string; badge?: number }> = [
    { id: 'condition', label: 'Условие' },
    { id: 'code', label: 'Код' },
    { id: 'result', label: 'Тесты', badge: failedCount },
  ]

  function revealHint() {
    setOpenedHints((n) => Math.min(n + 1, hintsCount))
  }

  function resetCode() {
    if (!window.confirm('Сбросить код к заготовке? Ваши изменения будут потеряны.')) return
    setCode(task.starter_code)
  }

  async function runGo() {
    setRunning(true)
    setGoResult(null)
    try {
      const res = await api.runCoding(slug, code)
      setGoResult(res)
      setActiveTab('result')
      if (res.result === 'passed') {
        try {
          const fresh = await api.getCodingTask(slug)
          setTask(fresh)
          toast.success(`Решено! Следующее перерешивание: ${formatDate(fresh.due_at)}`)
        } catch {
          toast.success('Решено!')
        }
      }
    } catch (e) {
      if (e instanceof ApiError && e.status === 503) {
        toast.error('Раннер занят, попробуйте ещё раз')
      } else if (e instanceof ApiError && e.status === 400) {
        toast.error('Код слишком большой')
      } else {
        toast.error(e instanceof ApiError ? e.message : 'Не удалось запустить тесты')
      }
    } finally {
      setRunning(false)
    }
  }

  async function runSql() {
    if (!task.schema_sql || !task.expected) return
    setRunning(true)
    setSqlOutcome(null)
    try {
      const { runSqlQuery } = await import('@/lib/sqlRunner')
      const result = await runSqlQuery(task.schema_sql, task.seed_sql ?? '', code)
      const diff = compareRows(result.rows, task.expected.rows, task.expected.order_matters)
      setSqlOutcome({ result, diff })
      setActiveTab('result')
      if (diff.matched) {
        try {
          await api.solveCoding(slug, code)
          const fresh = await api.getCodingTask(slug)
          setTask(fresh)
          toast.success(`Решено! Следующее перерешивание: ${formatDate(fresh.due_at)}`)
        } catch (e) {
          toast.error(e instanceof ApiError ? e.message : 'Не удалось сохранить решение')
        }
      }
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e)
      setSqlOutcome({ error: message })
      setActiveTab('result')
    } finally {
      setRunning(false)
    }
  }

  function handleRun() {
    if (running) return
    if (task.kind === 'go') void runGo()
    else void runSql()
  }

  async function openSolution() {
    const ok = window.confirm(
      'Разбор откроется, задача останется нерешённой — после разбора напишите решение по памяти.',
    )
    if (!ok) return
    setSolutionLoading(true)
    try {
      const res = await api.giveupCoding(slug)
      setSolutionMd(res.solution_md)
      setTask({
        ...task,
        gave_up: true,
        status: task.status === 'new' ? 'attempted' : task.status,
      })
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        try {
          const s = await api.getCodingSolution(slug)
          setSolutionMd(s.solution_md)
        } catch {
          toast.error('Не удалось открыть разбор')
        }
      } else {
        toast.error(e instanceof ApiError ? e.message : 'Не удалось открыть разбор')
      }
    } finally {
      setSolutionLoading(false)
    }
  }

  async function showSolution() {
    setSolutionLoading(true)
    try {
      const s = await api.getCodingSolution(slug)
      setSolutionMd(s.solution_md)
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Не удалось загрузить разбор')
    } finally {
      setSolutionLoading(false)
    }
  }

  const statementStrip = task.statement_md.replace(/[#>*`_-]/g, '').trim().slice(0, 110)

  return (
    <div className="mx-auto max-w-[1200px] pb-24">
      {/* Мобильные вкладки */}
      <div className="mb-6 [@media(min-width:900px)]:hidden">
        <div className="flex gap-1.5" role="tablist" aria-label="Разделы задачи">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={activeTab === tab.id}
              onClick={() => setActiveTab(tab.id)}
              style={{ fontFamily: FONT_UI }}
              className={cn(
                'flex-1 rounded-full border px-0 py-1.5 text-[13px] font-medium transition-colors',
                activeTab === tab.id
                  ? 'border-[var(--accent)] bg-[var(--accent)] text-[var(--bg)]'
                  : 'border-[var(--hairline)] text-[var(--ink-2)] hover:text-[var(--ink)]',
              )}
            >
              {tab.label}
              {tab.badge ? <span className="text-[var(--err)]"> ·{tab.badge}</span> : null}
            </button>
          ))}
        </div>
      </div>

      <div className="[@media(min-width:900px)]:grid [@media(min-width:900px)]:grid-cols-[minmax(0,400px)_minmax(0,1fr)] [@media(min-width:900px)]:items-start [@media(min-width:900px)]:gap-x-8">
        {/* ---------- Условие ---------- */}
        <section
          className={cn(
            panelClasses(activeTab, 'condition'),
            '[@media(min-width:900px)]:border-r [@media(min-width:900px)]:border-[var(--hairline)] [@media(min-width:900px)]:pr-8',
          )}
        >
          <Link
            to="/code"
            style={{ fontFamily: FONT_UI }}
            className="mb-4 inline-block text-[13px] text-[var(--ink-2)] transition-colors hover:text-[var(--ink)]"
          >
            ← Лайвкодинг
          </Link>

          <div className="mb-2.5 flex flex-wrap items-center gap-2">
            <DifficultyBadge difficulty={task.difficulty} />
            <span
              style={{ fontFamily: FONT_UI }}
              className="inline-flex items-center rounded-full bg-[var(--accent-soft)] px-2.5 py-[3px] text-[11.5px] font-medium text-[var(--ink-2)]"
            >
              {task.kind === 'go' ? 'Go' : 'SQL'}
            </span>
            {task.tags.length > 0 && (
              <span style={{ fontFamily: FONT_UI }} className="text-[13px] text-[var(--ink-3)]">
                {task.tags.map((t) => `#${t}`).join(' ')}
              </span>
            )}
          </div>

          <h1
            style={{ fontFamily: FONT_CONTENT }}
            className="mb-5 text-[26px] leading-[1.3] font-bold text-[var(--ink)] [text-wrap:pretty]"
          >
            {task.title}
          </h1>

          {/* Таймер 25/5 */}
          <div className="mb-6">
            <TimerPill timer={timer} />
          </div>

          {/* Свёртка условия в полосу после 25:00 (1j) */}
          {conditionCollapsed ? (
            <button
              type="button"
              onClick={() => setConditionCollapsed(false)}
              style={{ fontFamily: FONT_UI }}
              className="flex w-full items-center justify-between gap-3 rounded-[var(--r-md)] bg-[var(--accent-soft)] px-4 py-3 text-left text-[13px] text-[var(--ink-2)]"
            >
              <span className="truncate">{statementStrip}…</span>
              <span className="shrink-0 font-medium text-[var(--ink)]">Развернуть ⌄</span>
            </button>
          ) : (
            <>
              {timer.isTimeUp && (
                <button
                  type="button"
                  onClick={() => setConditionCollapsed(true)}
                  style={{ fontFamily: FONT_UI }}
                  className="mb-3 hidden text-[12px] text-[var(--ink-3)] transition-colors hover:text-[var(--ink)] [@media(min-width:900px)]:block"
                >
                  Свернуть условие ⌃
                </button>
              )}

              <Markdown size="sm">{task.statement_md}</Markdown>

              {task.kind === 'sql' && (
                <div className="mt-6 space-y-4">
                  {task.expected && (
                    <div>
                      <p
                        style={{ fontFamily: FONT_UI }}
                        className="mb-1.5 text-[13px] text-[var(--ink-2)]"
                      >
                        Ожидаемый результат:
                      </p>
                      <DataTable columns={task.expected.columns} rows={task.expected.rows} />
                    </div>
                  )}
                  {(task.schema_sql || task.seed_sql) && (
                    <details className="border-t border-[var(--hairline)] pt-4">
                      <summary
                        style={{ fontFamily: FONT_CODE }}
                        className="cursor-pointer text-[11px] font-semibold tracking-[0.08em] text-[var(--ink-3)] uppercase"
                      >
                        Схема и данные
                      </summary>
                      <pre
                        style={{ fontFamily: FONT_CODE }}
                        className="mt-3 overflow-x-auto rounded-[var(--r-md)] bg-[var(--accent-soft)] p-3 text-[13px] leading-relaxed text-[var(--ink)]"
                      >
                        {task.schema_sql}
                        {task.seed_sql ? `\n${task.seed_sql}` : ''}
                      </pre>
                    </details>
                  )}
                </div>
              )}
            </>
          )}

          {/* Подсказки */}
          <div className="mt-7 border-t border-[var(--hairline)] pt-6">
            <div className="mb-3.5 flex items-center justify-between">
              <span
                style={{ fontFamily: FONT_UI }}
                className="text-[13px] font-semibold text-[var(--ink)]"
              >
                Подсказки
              </span>
              <span style={{ fontFamily: FONT_CODE }} className="text-[11px] text-[var(--ink-3)]">
                {openedHints} / {hintsCount} открыто
              </span>
            </div>

            {openedHints > 0 && (
              <div className="mb-4 space-y-3">
                {task.hints.slice(0, openedHints).map((hint, i) => (
                  <div
                    key={i}
                    className="border-l-2 border-[var(--accent)] pl-3"
                    style={{ fontFamily: FONT_UI }}
                  >
                    <Markdown size="sm">{hint}</Markdown>
                  </div>
                ))}
              </div>
            )}

            {openedHints < hintsCount && (
              <div className="space-y-2.5">
                <button
                  type="button"
                  onClick={revealHint}
                  style={{ fontFamily: FONT_UI }}
                  className={cn(
                    'rounded-full border px-[18px] py-2 text-[14px] font-medium transition-colors',
                    timer.nudgeHint === openedHints + 1
                      ? 'border-[var(--warn)] bg-[var(--warn-soft)] text-[var(--warn)]'
                      : 'border-[var(--accent)] text-[var(--accent)] hover:bg-[var(--accent-soft)]',
                  )}
                >
                  Показать подсказку {openedHints + 1}
                </button>
                <div
                  style={{ fontFamily: FONT_UI }}
                  className="flex items-center gap-2 text-[12px] text-[var(--ink-3)]"
                >
                  <span className="size-[7px] shrink-0 rounded-full bg-[var(--hairline)]" />
                  {timer.nextHintInSeconds !== null
                    ? `Подсказка ${openedHints + 1} — через ${formatHintCountdown(timer.nextHintInSeconds)}`
                    : `Подсказка ${openedHints + 1} готова к открытию`}
                </div>
                <div
                  style={{ fontFamily: FONT_UI }}
                  className="flex items-center gap-2 text-[12px] text-[#C9C3B7]"
                >
                  <span className="size-[7px] shrink-0 rounded-full bg-[var(--hairline)]" />
                  Открыть разбор — после всех подсказок
                </div>
              </div>
            )}

            {canGiveUp && allHintsOpen && (
              <div className="mt-4 space-y-2">
                <button
                  type="button"
                  onClick={openSolution}
                  disabled={solutionLoading}
                  style={{ fontFamily: FONT_UI }}
                  className="rounded-full border border-[var(--accent)] px-[18px] py-2 text-[14px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)] disabled:opacity-60"
                >
                  {solutionLoading ? 'Открываем…' : 'Открыть разбор'}
                </button>
                <p style={{ fontFamily: FONT_UI }} className="text-[13px] text-[var(--ink-3)]">
                  Разберите решение и напишите код по памяти.
                </p>
              </div>
            )}

            {(solved || task.gave_up) && !solutionMd && (
              <button
                type="button"
                onClick={showSolution}
                disabled={solutionLoading}
                style={{ fontFamily: FONT_UI }}
                className="mt-4 rounded-full border border-[var(--accent)] px-[18px] py-2 text-[14px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)] disabled:opacity-60"
              >
                {solutionLoading ? 'Загружаем…' : 'Показать разбор'}
              </button>
            )}
          </div>

          {/* Разбор */}
          {solutionMd && (
            <div className="mt-7 border-t border-[var(--hairline)] pt-6">
              <SectionHead>Разбор</SectionHead>
              {!solved && (
                <div
                  style={{ fontFamily: FONT_UI }}
                  className="mb-4 rounded-[var(--r-md)] border border-[var(--accent)]/30 bg-[var(--accent-soft)] px-4 py-3 text-[13px] text-[var(--ink)]"
                >
                  Теперь напишите решение по памяти и прогоните тесты.
                </div>
              )}
              <Markdown size="sm">{solutionMd}</Markdown>
              <p
                style={{ fontFamily: FONT_UI }}
                className="mt-4 rounded-[var(--r-md)] border border-[var(--hairline)] bg-[var(--surface)] px-4 py-3 text-[12px] leading-[1.6] text-[var(--ink-2)]"
              >
                После разбора задача вернётся через{' '}
                <b className="text-[var(--ink)]">7 → 21 → 60 дней</b> для решения по памяти.
              </p>
            </div>
          )}
        </section>

        {/* ---------- Код + Тесты ---------- */}
        <div className="[@media(min-width:900px)]:min-w-0">
          <section className={panelClasses(activeTab, 'code')}>
            <div
              className={cn(
                'overflow-hidden rounded-[var(--r-md)] ring-1',
                solved ? 'ring-[var(--ok)]/50' : 'ring-[var(--editor-line)]',
              )}
            >
              <div className="flex items-center justify-between gap-2 border-b border-[var(--editor-line)] bg-[var(--editor)] px-4">
                <div
                  style={{ fontFamily: FONT_CODE }}
                  className="flex items-center gap-2 border-b-2 border-[var(--editor-ink)] py-2.5"
                >
                  <span className="text-[12px] font-medium text-[var(--editor-ink)]">{fileName}</span>
                  {dirty && (
                    <span
                      className="size-1.5 rounded-full bg-[var(--editor-ink)]"
                      aria-label="Есть несохранённые изменения"
                    />
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <span
                    style={{ fontFamily: FONT_CODE }}
                    className="hidden text-[11px] text-[#5C6167] [@media(min-width:900px)]:inline"
                  >
                    ⌘↩
                  </span>
                  <button
                    type="button"
                    onClick={handleRun}
                    disabled={running}
                    aria-label="Запустить"
                    className="flex size-7 shrink-0 items-center justify-center rounded-full border border-white/10 text-[var(--editor-ink)] transition-colors hover:bg-[var(--editor-ink)] hover:text-[var(--editor)] disabled:pointer-events-none disabled:opacity-60"
                  >
                    {running ? (
                      <Loader2Icon className="size-3.5 animate-spin" />
                    ) : (
                      <PlayIcon className="size-3 fill-current" />
                    )}
                  </button>
                </div>
              </div>
              <CodeEditor value={code} onChange={setCode} language={task.kind} onRun={handleRun} />
            </div>

            <div className="mt-4">
              <button
                type="button"
                onClick={resetCode}
                disabled={running}
                style={{ fontFamily: FONT_UI }}
                className="rounded-full border border-[var(--accent)] px-[18px] py-2 text-[14px] font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent-soft)] disabled:opacity-60"
              >
                Сбросить к заготовке
              </button>
            </div>
          </section>

          <section
            className={cn(
              panelClasses(activeTab, 'result'),
              '[@media(min-width:900px)]:mt-5 [@media(min-width:900px)]:border-t [@media(min-width:900px)]:border-[var(--hairline)] [@media(min-width:900px)]:pt-5',
            )}
          >
            <div className="hidden [@media(min-width:900px)]:block">
              <SectionHead>Результат</SectionHead>
            </div>
            {task.kind === 'go' ? (
              <>
                <GoConsole result={goResult} />
                {goResult && <GoRunSummary result={goResult} />}
              </>
            ) : (
              <SqlResultView outcome={sqlOutcome} />
            )}
          </section>
        </div>
      </div>

      {/* Нижняя закреплённая панель — мобайл (1k) */}
      <div className="sticky bottom-0 z-20 mt-6 flex items-center gap-3 border-t border-[var(--hairline)] bg-[var(--bg)] py-3 [@media(min-width:900px)]:hidden">
        <span style={{ fontFamily: FONT_UI }} className="min-w-0 flex-1 truncate text-[12px]">
          {timer.nudgeHint !== null ? (
            <span className="text-[var(--warn)]">Пора открыть подсказку {timer.nudgeHint}</span>
          ) : timer.nextHintInSeconds !== null ? (
            <span className="text-[var(--ink-3)]">
              Подсказка {openedHints + 1} через {formatHintCountdown(timer.nextHintInSeconds)}
            </span>
          ) : (
            <span className="text-[var(--ink-3)]">
              {allHintsOpen ? 'Все подсказки открыты' : 'Решайте, подсказки — по одной'}
            </span>
          )}
        </span>
        <button
          type="button"
          onClick={handleRun}
          disabled={running}
          style={{ fontFamily: FONT_UI }}
          className="shrink-0 rounded-full bg-[var(--accent)] px-5 py-2.5 text-[13px] font-medium text-[var(--bg)] transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-60"
        >
          {running ? busyLabel : `▸ ${runLabel}`}
        </button>
      </div>
    </div>
  )
}

function TimerPill({ timer }: { timer: ReturnType<typeof useAttemptTimer> }) {
  if (timer.phase === 'over') {
    return (
      <span
        style={{ fontFamily: FONT_CODE }}
        className="inline-flex items-center gap-2.5 rounded-full bg-[var(--warn-soft)] px-3.5 py-1.5"
        aria-label="Таймер попытки — сверх времени"
      >
        <span className="size-2 shrink-0 rounded-full bg-[var(--warn)]" />
        <span className="text-[16px] font-semibold tabular-nums text-[#7A5A17]">{timer.display}</span>
        {timer.nextHintInSeconds !== null && (
          <span style={{ fontFamily: FONT_UI }} className="text-[11px] text-[var(--warn)]">
            подсказка через {formatHintCountdown(timer.nextHintInSeconds)}
          </span>
        )}
      </span>
    )
  }
  return (
    <span
      style={{ fontFamily: FONT_CODE }}
      className="inline-flex items-center gap-2.5 rounded-full border border-[var(--hairline)] bg-[var(--surface)] px-3.5 py-1.5"
      aria-label="Таймер попытки — фокус"
    >
      <span className="size-2 shrink-0 rounded-full bg-[var(--ok)]" />
      <span className="text-[16px] font-semibold tabular-nums text-[var(--ink)]">{timer.display}</span>
      <span style={{ fontFamily: FONT_UI }} className="text-[11px] text-[var(--ink-3)]">
        фокус
      </span>
    </span>
  )
}
