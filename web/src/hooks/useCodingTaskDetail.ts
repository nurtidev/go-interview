import { useCallback, useEffect, useState } from 'react'
import { ApiError, api, type CodingTaskDetail } from '@/lib/api'

interface UseCodingTaskDetailResult {
  data: CodingTaskDetail | null
  loading: boolean
  error: string | null
  reload: () => void
  setData: (data: CodingTaskDetail) => void
}

export function useCodingTaskDetail(slug: string | undefined): UseCodingTaskDetailResult {
  const [data, setData] = useState<CodingTaskDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!slug) return
    setLoading(true)
    setError(null)
    api
      .getCodingTask(slug)
      .then(setData)
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Не удалось загрузить задачу')
      })
      .finally(() => setLoading(false))
  }, [slug])

  useEffect(() => {
    load()
  }, [load])

  return { data, loading, error, reload: load, setData }
}
