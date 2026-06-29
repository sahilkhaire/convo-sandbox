import { useQuery } from '@tanstack/react-query'
import { api } from '../api'

export default function WebhooksPage() {
  const { data: webhooks = [], isLoading } = useQuery({
    queryKey: ['webhooks'],
    queryFn: () => api.listWebhooks(100),
    refetchInterval: 5000,
  })

  return (
    <div className="h-full flex flex-col">
      <header className="px-6 py-4 border-b border-slate-800">
        <h2 className="text-lg font-medium">Webhook Log</h2>
        <p className="text-sm text-slate-500">Outbound webhook deliveries to your platform</p>
      </header>
      <div className="flex-1 overflow-y-auto p-6 space-y-3">
        {isLoading && <p className="text-slate-500">Loading...</p>}
        {!isLoading && webhooks.length === 0 && (
          <p className="text-slate-500">No webhook deliveries yet.</p>
        )}
        {webhooks.map((w) => (
          <div key={w.id} className="p-4 bg-slate-900/50 rounded-xl border border-slate-800">
            <div className="flex items-center gap-3 text-sm">
              <span
                className={`uppercase text-xs px-2 py-0.5 rounded ${
                  w.channel === 'rcs' ? 'bg-blue-900 text-blue-300' : 'bg-green-900 text-green-300'
                }`}
              >
                {w.channel}
              </span>
              <span className="text-slate-300">{w.event_type}</span>
              <span
                className={`ml-auto ${
                  w.http_status && w.http_status >= 200 && w.http_status < 300
                    ? 'text-emerald-400'
                    : 'text-red-400'
                }`}
              >
                {w.http_status ?? '—'}
              </span>
              <span className="text-slate-600 text-xs">
                {new Date(w.created_at).toLocaleString()}
              </span>
            </div>
            {w.error_message && (
              <p className="text-red-400 text-xs mt-2">{w.error_message}</p>
            )}
            <pre className="mt-3 text-xs text-slate-400 overflow-x-auto bg-slate-950 p-3 rounded-lg max-h-48">
              {JSON.stringify(w.payload, null, 2)}
            </pre>
          </div>
        ))}
      </div>
    </div>
  )
}
