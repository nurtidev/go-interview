import { useCallback, useEffect, useState } from 'react'
import { ApiError, api, type QuestionDetail } from '@/lib/api'

interface UseQuestionDetailResult {
  data: QuestionDetail | null
  loading: boolean
  error: string | null
  reload: () => void
}

export function useQuestionDetail(slug: string | undefined): UseQuestionDetailResult {
  const [data, setData] = useState<QuestionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!slug) return
    setLoading(true)
    setError(null)
    api
      .getQuestion(slug)
      .then(setData)
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить вопрос')
      })
      .finally(() => setLoading(false))
  }, [slug])

  useEffect(() => {
    load()
  }, [load])

  return { data, loading, error, reload: load }
}
