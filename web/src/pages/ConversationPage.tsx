import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, messagePreview } from '../api'
import { useAuth } from '../auth/AuthContext'

export default function ConversationPage() {
  const { can } = useAuth()
  const canReply = can('action_reply')
  const canDelivery = can('action_delivery')
  const { id } = useParams<{ id: string }>()
  const [text, setText] = useState('')
  const qc = useQueryClient()

  const { data: conversation } = useQuery({
    queryKey: ['conversation', id],
    queryFn: () => api.getConversation(id!),
    enabled: !!id,
  })

  const { data: messages = [] } = useQuery({
    queryKey: ['messages', id],
    queryFn: () => api.listMessages(id!),
    enabled: !!id,
    refetchInterval: 3000,
  })

  const send = useMutation({
    mutationFn: (t: string) => api.sendInbound(id!, t),
    onSuccess: () => {
      setText('')
      qc.invalidateQueries({ queryKey: ['messages', id] })
      qc.invalidateQueries({ queryKey: ['conversations'] })
    },
  })

  const status = useMutation({
    mutationFn: ({ msgId, s }: { msgId: string; s: string }) => api.triggerStatus(msgId, s),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['messages', id] }),
  })

  if (!id) return null

  return (
    <div className="h-full flex">
      <div className="flex-1 flex flex-col">
        <header className="px-6 py-4 border-b border-slate-800">
          <h2 className="font-medium">{conversation?.user_phone}</h2>
          <p className="text-sm text-slate-500">
            {conversation?.channel?.toUpperCase()} · {conversation?.account_name} (
            {conversation?.account_external_id})
          </p>
        </header>

        <div className="flex-1 overflow-y-auto p-6 space-y-3">
          {messages.map((m) => (
            <div
              key={m.id}
              className={`flex ${m.direction === 'inbound' ? 'justify-start' : 'justify-end'}`}
            >
              <div
                className={`max-w-[70%] rounded-2xl px-4 py-2 ${
                  m.direction === 'inbound'
                    ? 'bg-slate-800 rounded-bl-sm'
                    : 'bg-emerald-900/60 rounded-br-sm'
                }`}
              >
                <p className="text-sm whitespace-pre-wrap">{messagePreview(m.body)}</p>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs text-slate-500">{m.status}</span>
                  <span className="text-xs text-slate-600">
                    {new Date(m.created_at).toLocaleTimeString()}
                  </span>
                </div>
                {m.direction === 'outbound' && canDelivery && (
                  <div className="flex gap-1 mt-2 flex-wrap">
                    {['delivered', 'read', 'failed', 'revoked'].map((s) => (
                      <button
                        key={s}
                        onClick={() => status.mutate({ msgId: m.id, s })}
                        className="text-xs px-2 py-0.5 rounded bg-slate-700 hover:bg-slate-600"
                      >
                        {s}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>

        {canReply ? (
          <form
            className="p-4 border-t border-slate-800 flex gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              if (text.trim()) send.mutate(text.trim())
            }}
          >
            <input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Reply as end-user (triggers inbound webhook)..."
              className="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-4 py-2 text-sm focus:outline-none focus:border-emerald-600"
            />
            <button
              type="submit"
              disabled={send.isPending || !text.trim()}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 rounded-lg text-sm font-medium"
            >
              Send
            </button>
          </form>
        ) : (
          <div className="p-4 border-t border-slate-800 text-sm text-slate-500">
            You do not have permission to send replies.
          </div>
        )}
      </div>

      <aside className="w-72 border-l border-slate-800 p-4 overflow-y-auto">
        <h3 className="text-sm font-medium text-slate-400 mb-3">Delivery Panel</h3>
        <p className="text-xs text-slate-500 mb-4">
          {canDelivery
            ? 'Click status buttons on outbound messages to manually fire delivery webhooks to your platform.'
            : 'Delivery controls require the action_delivery permission.'}
        </p>
        <div className="space-y-2 text-xs">
          {messages
            .filter((m) => m.direction === 'outbound')
            .slice(-5)
            .reverse()
            .map((m) => (
              <div key={m.id} className="p-2 bg-slate-900 rounded-lg">
                <p className="truncate text-slate-300">{messagePreview(m.body)}</p>
                <p className="text-slate-600 mt-1 font-mono truncate">{m.vendor_message_id}</p>
                <p className="text-emerald-500 mt-1">{m.status}</p>
              </div>
            ))}
        </div>
      </aside>
    </div>
  )
}
