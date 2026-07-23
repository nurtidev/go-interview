// Типизированный клиент для backend API GoPrep.
// Контракт зафиксирован в задаче — см. README/CLAUDE.md проекта.

export type Difficulty = 'middle' | 'senior' | 'staff'
export type QuestionStatus = 'new' | 'learning' | 'review'
export type AnswerLevelName = 'middle' | 'senior' | 'deep'
export type Grade = 'again' | 'hard' | 'good' | 'easy'

export interface User {
  id: number
  email: string
}

/** Профиль пользователя (GET/PATCH /api/me). Поля name/interview_date опциональны. */
export interface Me {
  id: number
  email: string
  name: string | null
  interview_date: string | null
}

export interface UpdateMePayload {
  name?: string | null
  interview_date?: string | null
}

export interface AuthResponse {
  token: string
  user: User
}

/** Публичные, несекретные флаги окружения (GET /api/config, без auth). */
export interface ConfigResponse {
  registration_enabled: boolean
}

export interface Section {
  id: string
  title: string
  description: string
  total: number
  done: number
  due: number
}

export interface SectionsResponse {
  sections: Section[]
}

export interface QuestionListItem {
  slug: string
  title: string
  difficulty: Difficulty
  tags: string[]
  status: QuestionStatus
  due_at: string | null
}

export interface QuestionsResponse {
  questions: QuestionListItem[]
}

export interface AnswerLevel {
  level: AnswerLevelName
  text_md: string
}

export interface QuestionDetail {
  slug: string
  section: string
  title: string
  difficulty: Difficulty
  tags: string[]
  question_md: string
  answer_levels: AnswerLevel[]
  follow_ups: string[]
  status: QuestionStatus
  due_at: string | null
}

export interface ReviewResult {
  status: QuestionStatus
  due_at: string | null
  interval_days: number
}

export interface ReviewQueueItem {
  slug: string
  title: string
  section: string
  due_at: string | null
}

export interface ReviewQueueResponse {
  questions: ReviewQueueItem[]
}

export interface SectionStat {
  id: string
  title: string
  total: number
  done: number
}

export interface LessonStats {
  total: number
  read: number
}

export interface CodingStats {
  total: number
  solved: number
  due: number
}

/** Одна точка активности для heatmap (день + число событий). */
export interface ActivityDay {
  date: string
  count: number
}

export interface Stats {
  total: number
  reviewed: number
  due_today: number
  streak_days: number
  by_section: SectionStat[]
  /** Может отсутствовать, пока бэкенд не начал отдавать статистику по учебнику. */
  lessons?: LessonStats
  /** Может отсутствовать, пока бэкенд не начал отдавать статистику по лайвкодингу. */
  coding?: CodingStats
  /** Активность по дням (≈84 дня) для heatmap. Может отсутствовать — тогда блок скрыт. */
  activity?: ActivityDay[]
  /** Рекорд серии. Может отсутствовать. */
  streak_record?: number
}

export type LessonTopic = 'go-internals' | 'concurrency' | 'networks' | 'os'

export interface LessonListItem {
  slug: string
  topic: LessonTopic
  title: string
  minutes: number
  tags: string[]
  read: boolean
  reinforce_done: number
  reinforce_total: number
}

export interface LessonsResponse {
  lessons: LessonListItem[]
}

export type LessonRelatedType = 'question' | 'task'

export interface LessonRelatedItem {
  type: LessonRelatedType
  slug: string
  title: string
  status: string
}

export interface LessonDetail {
  slug: string
  topic: LessonTopic
  title: string
  minutes: number
  tags: string[]
  body_md: string
  read: boolean
  related: LessonRelatedItem[]
}

export interface MarkLessonReadResponse {
  read: true
}

// ---------- Лайвкодинг ----------

export type CodingKind = 'go' | 'sql'
export type CodingStatus = 'new' | 'attempted' | 'solved'

export interface CodingTaskListItem {
  slug: string
  kind: CodingKind
  title: string
  difficulty: Difficulty
  tags: string[]
  status: CodingStatus
  gave_up: boolean
  due_at: string | null
  due: boolean
}

export interface CodingTasksResponse {
  tasks: CodingTaskListItem[]
}

export interface SqlExpected {
  columns: string[]
  rows: string[][]
  order_matters: boolean
}

export interface CodingTaskDetail {
  slug: string
  kind: CodingKind
  title: string
  difficulty: Difficulty
  tags: string[]
  statement_md: string
  starter_code: string
  hints: string[]
  status: CodingStatus
  gave_up: boolean
  due_at: string | null
  last_code: string | null
  solution_md_available: boolean
  /** Только для sql-задач. */
  schema_sql?: string
  seed_sql?: string
  expected?: SqlExpected
}

export type RunResultKind = 'passed' | 'tests_failed' | 'compile_error' | 'timeout'

export interface RunResponse {
  result: RunResultKind
  output: string
}

export interface SolvedResponse {
  status: 'solved'
}

export interface GiveupResponse {
  solution_md: string
}

export interface SolutionResponse {
  solution_md: string
}

const TOKEN_KEY = 'goprep_token'
const USER_KEY = 'goprep_user'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getStoredUser(): User | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function setStoredAuth(token: string, user: User): void {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearStoredAuth(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

/** Событие: сервер ответил 401 на аутентифицированный запрос — сессия истекла. */
export const AUTH_EXPIRED_EVENT = 'goprep:auth-expired'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  /** Прикреплять ли Authorization-заголовок и реагировать на 401 как на протухшую сессию. */
  auth?: boolean
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, auth = true } = opts
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (auth && token) {
    headers.Authorization = `Bearer ${token}`
  }

  let res: Response
  try {
    res = await fetch(`/api${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError('Не удалось соединиться с сервером. Проверьте подключение.', 0)
  }

  if (res.status === 401 && auth && token) {
    clearStoredAuth()
    window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT))
  }

  if (!res.ok) {
    let message = 'Что-то пошло не так. Попробуйте ещё раз.'
    try {
      const data: unknown = await res.json()
      if (data && typeof data === 'object' && 'error' in data && typeof data.error === 'string') {
        message = data.error
      }
    } catch {
      // ответ не JSON — используем сообщение по умолчанию
    }
    throw new ApiError(message, res.status)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  getConfig: () => request<ConfigResponse>('/config', { auth: false }),

  register: (email: string, password: string) =>
    request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: { email, password },
      auth: false,
    }),

  login: (email: string, password: string) =>
    request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: { email, password },
      auth: false,
    }),

  getMe: () => request<Me>('/me'),

  updateMe: (payload: UpdateMePayload) =>
    request<Me>('/me', { method: 'PATCH', body: payload }),

  getSections: () => request<SectionsResponse>('/sections'),

  getQuestions: (sectionId: string) =>
    request<QuestionsResponse>(`/questions?section=${encodeURIComponent(sectionId)}`),

  getQuestion: (slug: string) => request<QuestionDetail>(`/questions/${encodeURIComponent(slug)}`),

  submitReview: (slug: string, grade: Grade) =>
    request<ReviewResult>(`/review/${encodeURIComponent(slug)}`, {
      method: 'POST',
      body: { grade },
    }),

  getReviewQueue: () => request<ReviewQueueResponse>('/review/queue'),

  getStats: () => request<Stats>('/me/stats'),

  getLessons: () => request<LessonsResponse>('/lessons'),

  getLesson: (slug: string) => request<LessonDetail>(`/lessons/${encodeURIComponent(slug)}`),

  markLessonRead: (slug: string) =>
    request<MarkLessonReadResponse>(`/lessons/${encodeURIComponent(slug)}/read`, {
      method: 'POST',
    }),

  getCodingTasks: () => request<CodingTasksResponse>('/coding/tasks'),

  getCodingTask: (slug: string) =>
    request<CodingTaskDetail>(`/coding/tasks/${encodeURIComponent(slug)}`),

  runCoding: (slug: string, code: string) =>
    request<RunResponse>(`/coding/tasks/${encodeURIComponent(slug)}/run`, {
      method: 'POST',
      body: { code },
    }),

  solveCoding: (slug: string, code: string) =>
    request<SolvedResponse>(`/coding/tasks/${encodeURIComponent(slug)}/solved`, {
      method: 'POST',
      body: { code },
    }),

  giveupCoding: (slug: string) =>
    request<GiveupResponse>(`/coding/tasks/${encodeURIComponent(slug)}/giveup`, {
      method: 'POST',
    }),

  getCodingSolution: (slug: string) =>
    request<SolutionResponse>(`/coding/tasks/${encodeURIComponent(slug)}/solution`),
}
