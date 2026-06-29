import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Account } from '../api'
import { useAuth } from '../auth/AuthContext'

const emptyForm: Partial<Account> = {
  channel: 'rcs',
  name: '',
  external_id: '',
  client_secret: '',
  access_token: '',
  webhook_url: '',
  webhook_verify_token: 'verify_token',
  waba_id: '',
  display_phone: '',
  sent_delay_ms: 200,
  delivered_delay_ms: 800,
  read_delay_ms: 3000,
  auto_read: false,
}

export default function AccountsPage() {
  const { can } = useAuth()
  const canWrite = can('action_accounts_write')
  const [form, setForm] = useState<Partial<Account>>(emptyForm)
  const [editing, setEditing] = useState<string | null>(null)
  const qc = useQueryClient()

  const { data: accounts = [] } = useQuery({
    queryKey: ['accounts'],
    queryFn: api.listAccounts,
  })

  const create = useMutation({
    mutationFn: api.createAccount,
    onSuccess: () => {
      setForm(emptyForm)
      qc.invalidateQueries({ queryKey: ['accounts'] })
    },
  })

  const update = useMutation({
    mutationFn: ({ id, body }: { id: string; body: Partial<Account> }) => api.updateAccount(id, body),
    onSuccess: () => {
      setEditing(null)
      setForm(emptyForm)
      qc.invalidateQueries({ queryKey: ['accounts'] })
    },
  })

  const remove = useMutation({
    mutationFn: api.deleteAccount,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accounts'] }),
  })

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editing) {
      update.mutate({ id: editing, body: form })
    } else {
      create.mutate(form)
    }
  }

  return (
    <div className="h-full overflow-y-auto">
      <header className="px-6 py-4 border-b border-slate-800">
        <h2 className="text-lg font-medium">Accounts</h2>
        <p className="text-sm text-slate-500">RCS assistants and WhatsApp phone numbers</p>
      </header>

      <div className="p-6 grid lg:grid-cols-2 gap-8">
        {canWrite ? (
        <form onSubmit={submit} className="space-y-4 bg-slate-900/50 p-5 rounded-xl border border-slate-800">
          <h3 className="font-medium">{editing ? 'Edit Account' : 'New Account'}</h3>

          <label className="block text-sm">
            <span className="text-slate-400">Channel</span>
            <select
              value={form.channel}
              onChange={(e) => setForm({ ...form, channel: e.target.value as 'rcs' | 'whatsapp' })}
              className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
            >
              <option value="rcs">RCS (Jio)</option>
              <option value="whatsapp">WhatsApp (Meta)</option>
            </select>
          </label>

          <label className="block text-sm">
            <span className="text-slate-400">Name</span>
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
              required
            />
          </label>

          <label className="block text-sm">
            <span className="text-slate-400">
              {form.channel === 'rcs' ? 'Assistant ID (client_id)' : 'Phone Number ID'}
            </span>
            <input
              value={form.external_id}
              onChange={(e) => setForm({ ...form, external_id: e.target.value })}
              className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
              placeholder="Auto-generated if empty"
            />
          </label>

          {form.channel === 'rcs' && (
            <label className="block text-sm">
              <span className="text-slate-400">Client Secret</span>
              <input
                value={form.client_secret}
                onChange={(e) => setForm({ ...form, client_secret: e.target.value })}
                className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
              />
            </label>
          )}

          <label className="block text-sm">
            <span className="text-slate-400">Access Token</span>
            <input
              value={form.access_token}
              onChange={(e) => setForm({ ...form, access_token: e.target.value })}
              className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
              placeholder="Auto-generated if empty"
            />
          </label>

          <label className="block text-sm">
            <span className="text-slate-400">Webhook URL (deliveries go here)</span>
            <input
              value={form.webhook_url}
              onChange={(e) => setForm({ ...form, webhook_url: e.target.value })}
              className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
              placeholder="https://your-platform/webhooks/..."
              required
            />
          </label>

          {form.channel === 'whatsapp' && (
            <>
              <label className="block text-sm">
                <span className="text-slate-400">WABA ID</span>
                <input
                  value={form.waba_id}
                  onChange={(e) => setForm({ ...form, waba_id: e.target.value })}
                  className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
                />
              </label>
              <label className="block text-sm">
                <span className="text-slate-400">Display Phone</span>
                <input
                  value={form.display_phone}
                  onChange={(e) => setForm({ ...form, display_phone: e.target.value })}
                  className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2"
                />
              </label>
            </>
          )}

          <div className="grid grid-cols-3 gap-2">
            {(['sent_delay_ms', 'delivered_delay_ms', 'read_delay_ms'] as const).map((k) => (
              <label key={k} className="block text-sm">
                <span className="text-slate-400 text-xs">{k.replace(/_/g, ' ')}</span>
                <input
                  type="number"
                  value={form[k]}
                  onChange={(e) => setForm({ ...form, [k]: Number(e.target.value) })}
                  className="mt-1 w-full bg-slate-900 border border-slate-700 rounded-lg px-2 py-1 text-sm"
                />
              </label>
            ))}
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.auto_read}
              onChange={(e) => setForm({ ...form, auto_read: e.target.checked })}
            />
            Auto-send read receipts
          </label>

          <div className="flex gap-2">
            <button
              type="submit"
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-sm font-medium"
            >
              {editing ? 'Update' : 'Create'}
            </button>
            {editing && (
              <button
                type="button"
                onClick={() => {
                  setEditing(null)
                  setForm(emptyForm)
                }}
                className="px-4 py-2 bg-slate-700 rounded-lg text-sm"
              >
                Cancel
              </button>
            )}
          </div>
        </form>
        ) : (
          <div className="p-5 bg-slate-900/50 rounded-xl border border-slate-800 text-sm text-slate-500">
            You have read-only access to accounts. Contact an admin for write permissions.
          </div>
        )}

        <div className="space-y-3">
          {accounts.map((a) => (
            <div
              key={a.id}
              className="p-4 bg-slate-900/50 rounded-xl border border-slate-800 flex justify-between gap-4"
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span
                    className={`text-xs uppercase px-2 py-0.5 rounded ${
                      a.channel === 'rcs' ? 'bg-blue-900 text-blue-300' : 'bg-green-900 text-green-300'
                    }`}
                  >
                    {a.channel}
                  </span>
                  <span className="font-medium">{a.name}</span>
                </div>
                <p className="text-sm text-slate-500 mt-1 font-mono truncate">{a.external_id}</p>
                <p className="text-xs text-slate-600 mt-1 truncate">Webhook: {a.webhook_url}</p>
              </div>
              {canWrite && (
                <div className="flex flex-col gap-1 shrink-0">
                  <button
                    onClick={() => {
                      setEditing(a.id)
                      setForm(a)
                    }}
                    className="text-xs px-2 py-1 bg-slate-700 rounded hover:bg-slate-600"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => remove.mutate(a.id)}
                    className="text-xs px-2 py-1 bg-red-900/50 text-red-300 rounded hover:bg-red-900"
                  >
                    Delete
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
