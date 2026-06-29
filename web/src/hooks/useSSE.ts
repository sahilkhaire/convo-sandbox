import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api } from '../api'

export function useSSE() {
  const qc = useQueryClient()

  useEffect(() => {
    const token = api.getToken()
    if (!token) return

    const url = `${api.getApiBase()}/events?token=${encodeURIComponent(token)}`
    const es = new EventSource(url)
    es.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data)
        if (data.type === 'new_message' || data.type === 'delivery') {
          qc.invalidateQueries({ queryKey: ['conversations'] })
          qc.invalidateQueries({ queryKey: ['messages'] })
          qc.invalidateQueries({ queryKey: ['webhooks'] })
        }
        if (data.type === 'data_cleared') {
          qc.invalidateQueries()
        }
      } catch {
        /* ignore */
      }
    }
    return () => es.close()
  }, [qc])
}
