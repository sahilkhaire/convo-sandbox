import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../auth/AuthContext'

export default function SettingsPage() {
  const { can } = useAuth()
  const canPurge = can('action_data_purge')
  const [scope, setScope] = useState<'messages' | 'all'>('messages')
  const [confirmText, setConfirmText] = useState('')
  const [showModal, setShowModal] = useState(false)
  const qc = useQueryClient()
  const navigate = useNavigate()

  const purge = useMutation({
    mutationFn: () => api.purgeData(scope),
    onSuccess: () => {
      setShowModal(false)
      setConfirmText('')
      qc.invalidateQueries()
      navigate('/')
    },
  })

  const canConfirm = scope === 'messages' || confirmText === 'DELETE'

  return (
    <div className="h-full overflow-y-auto">
      <header className="px-6 py-4 border-b border-slate-800">
        <h2 className="text-lg font-medium">Settings</h2>
        <p className="text-sm text-slate-500">Simulator configuration and data management</p>
      </header>

      <div className="p-6 max-w-xl space-y-8">
        {canPurge && (
          <section className="p-5 bg-slate-900/50 rounded-xl border border-slate-800">
            <h3 className="font-medium text-red-400">Danger Zone</h3>
            <p className="text-sm text-slate-500 mt-2">
              Clear all simulator data from PostgreSQL. Use this to reset dev testing without
              redeploying.
            </p>
            <button
              onClick={() => setShowModal(true)}
              className="mt-4 px-4 py-2 bg-red-700 hover:bg-red-600 rounded-lg text-sm font-medium"
            >
              Clear all data
            </button>
          </section>
        )}

        <section className="p-5 bg-slate-900/50 rounded-xl border border-slate-800 text-sm text-slate-400 space-y-2">
          <h3 className="font-medium text-slate-200">Domain swap (dev)</h3>
          <pre className="bg-slate-950 p-3 rounded-lg text-xs overflow-x-auto">
{`JIO_OAUTH_BASE=http://localhost:8080
JIO_API_BASE=http://localhost:8080
WHATSAPP_API_BASE=http://localhost:8080

# Production-identical paths:
# GET  /v1/oauth/token
# POST /v1/messaging/users/{userPhoneNumber}/assistantMessages/async?messageId=&assistantId=
# POST /v19.0/{phone-number-id}/messages`}
          </pre>
        </section>
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center p-4 z-50">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 max-w-md w-full">
            <h3 className="text-lg font-medium">Clear all data?</h3>
            <p className="text-sm text-slate-400 mt-2">
              This will permanently delete data from PostgreSQL. This cannot be undone.
            </p>

            <div className="mt-4 space-y-2">
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="radio"
                  checked={scope === 'messages'}
                  onChange={() => setScope('messages')}
                />
                Clear messages only (keep accounts)
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="radio"
                  checked={scope === 'all'}
                  onChange={() => setScope('all')}
                />
                Clear everything (including accounts)
              </label>
            </div>

            {scope === 'all' && (
              <label className="block mt-4 text-sm">
                <span className="text-slate-400">Type DELETE to confirm</span>
                <input
                  value={confirmText}
                  onChange={(e) => setConfirmText(e.target.value)}
                  className="mt-1 w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2"
                  placeholder="DELETE"
                />
              </label>
            )}

            <div className="flex gap-2 mt-6">
              <button
                disabled={!canConfirm || purge.isPending}
                onClick={() => purge.mutate()}
                className="flex-1 px-4 py-2 bg-red-700 hover:bg-red-600 disabled:opacity-50 rounded-lg text-sm font-medium"
              >
                {purge.isPending ? 'Clearing...' : 'Confirm'}
              </button>
              <button
                onClick={() => {
                  setShowModal(false)
                  setConfirmText('')
                }}
                className="px-4 py-2 bg-slate-700 rounded-lg text-sm"
              >
                Cancel
              </button>
            </div>
            {purge.isError && (
              <p className="text-red-400 text-sm mt-2">{(purge.error as Error).message}</p>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
