import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api'

export default function InboxPage() {
  const { data: conversations = [], isLoading } = useQuery({
    queryKey: ['conversations'],
    queryFn: api.listConversations,
    refetchInterval: 5000,
  })

  if (isLoading) {
    return <div className="p-8 text-slate-400">Loading conversations...</div>
  }

  return (
    <div className="h-full flex flex-col">
      <header className="px-6 py-4 border-b border-slate-800">
        <h2 className="text-lg font-medium">Conversations</h2>
        <p className="text-sm text-slate-500">Grouped by channel, sender, and recipient</p>
      </header>
      <div className="flex-1 overflow-y-auto">
        {conversations.length === 0 ? (
          <div className="p-8 text-center text-slate-500">
            <p>No conversations yet.</p>
            <p className="text-sm mt-2">
              Send a message via the RCS or WhatsApp vendor API to create a thread.
            </p>
            <Link to="/accounts" className="text-emerald-400 text-sm mt-4 inline-block">
              Configure accounts →
            </Link>
          </div>
        ) : (
          <ul className="divide-y divide-slate-800">
            {conversations.map((c) => (
              <li key={c.id}>
                <Link
                  to={`/conversations/${c.id}`}
                  className="flex items-center gap-4 px-6 py-4 hover:bg-slate-900/80 transition-colors"
                >
                  <div
                    className={`w-10 h-10 rounded-full flex items-center justify-center text-xs font-bold uppercase ${
                      c.channel === 'rcs' ? 'bg-blue-900 text-blue-300' : 'bg-green-900 text-green-300'
                    }`}
                  >
                    {c.channel.slice(0, 2)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">{c.user_phone}</span>
                      {c.unread_count > 0 && (
                        <span className="bg-emerald-600 text-white text-xs px-1.5 py-0.5 rounded-full">
                          {c.unread_count}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-slate-500 truncate">
                      {c.account_name} · {c.account_external_id}
                    </p>
                    <p className="text-sm text-slate-400 truncate mt-0.5">
                      {c.last_message_preview || '—'}
                    </p>
                  </div>
                  <span className="text-xs text-slate-600 uppercase">{c.channel}</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
