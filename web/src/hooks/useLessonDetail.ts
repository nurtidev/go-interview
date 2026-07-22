import { useCallback, useEffect, useState } from 'react'
import { ApiError, api, type LessonDetail } from '@/lib/api'

interface UseLessonDetailResult {
  data: LessonDetail | null
  loading: boolean
  error: string | null
  reload: () => void
  setData: (data: LessonDetail) => void
}

export function useLessonDetail(slug: string | undefined): UseLessonDetailResult {
  const [data, setData] = useState<LessonDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!slug) return
    setLoading(true)
    setError(null)
    api
      .getLesson(slug)
      .then(setData)
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить главу')
      })
      .finally(() => setLoading(false))
  }, [slug])

  useEffect(() => {
    load()
  }, [load])

  return { data, loading, error, reload: load, setData }
}
