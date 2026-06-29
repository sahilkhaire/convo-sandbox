import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type AppUser } from '../api'
import { useAuth } from '../auth/AuthContext'
import { PERMISSIONS } from '../auth/types'

const VIEW_PERMS = [
  'view_inbox',
  'view_accounts',
  'view_webhooks',
  'view_settings',
  'view_users',
] as const

const ACTION_PERMS = [
  'action_reply',
  'action_delivery',
  'action_accounts_write',
  'action_data_purge',
  'action_users_manage',
] as const

function PermissionCheckboxes({
  selected,
  onChange,
  disabled,
}: {
  selected: string[]
  onChange: (perms: string[]) => void
  disabled?: boolean
}) {
  const toggle = (perm: string) => {
    if (disabled) return
    if (selected.includes(perm)) onChange(selected.filter((p) => p !== perm))
    else onChange([...selected, perm])
  }

  return (
    <div className="space-y-4">
      <div>
        <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">View</p>
        <div className="space-y-1">
          {VIEW_PERMS.map((p) => (
            <label key={p} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={selected.includes(p)}
                onChange={() => toggle(p)}
                disabled={disabled}
              />
              {PERMISSIONS[p]}
            </label>
          ))}
        </div>
      </div>
      <div>
        <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Actions</p>
        <div className="space-y-1">
          {ACTION_PERMS.map((p) => (
            <label key={p} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={selected.includes(p)}
                onChange={() => toggle(p)}
                disabled={disabled}
              />
              {PERMISSIONS[p]}
            </label>
          ))}
        </div>
      </div>
    </div>
  )
}

export default function UsersPage() {
  const { isAdmin, user: currentUser } = useAuth()
  const qc = useQueryClient()

  const [showCreate, setShowCreate] = useState(false)
  const [editing, setEditing] = useState<AppUser | null>(null)
  const [resetId, setResetId] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState('')

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [permissions, setPermissions] = useState<string[]>([])
  const [isAdminFlag, setIsAdminFlag] = useState(false)
  const [isActive, setIsActive] = useState(true)

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['users'],
    queryFn: api.listUsers,
    enabled: isAdmin,
  })

  const create = useMutation({
    mutationFn: () =>
      api.createUser({
        name,
        email,
        password,
        permissions,
        is_admin: isAdminFlag,
      }),
    onSuccess: () => {
      resetForm()
      setShowCreate(false)
      qc.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const update = useMutation({
    mutationFn: () =>
      api.updateUser(editing!.id, {
        name,
        is_active: isActive,
        permissions,
        is_admin: isAdminFlag,
      }),
    onSuccess: () => {
      resetForm()
      setEditing(null)
      qc.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const resetPwd = useMutation({
    mutationFn: () => api.resetUserPassword(resetId!, newPassword),
    onSuccess: () => {
      setResetId(null)
      setNewPassword('')
    },
  })

  const resetForm = () => {
    setName('')
    setEmail('')
    setPassword('')
    setPermissions([])
    setIsAdminFlag(false)
    setIsActive(true)
  }

  const startEdit = (u: AppUser) => {
    setEditing(u)
    setShowCreate(false)
    setName(u.name)
    setEmail(u.email)
    setPermissions(u.permissions)
    setIsAdminFlag(u.is_admin)
    setIsActive(u.is_active ?? true)
  }

  if (!isAdmin) return <Navigate to="/" replace />

  return (
    <div className="h-full overflow-y-auto">
      <header className="px-6 py-4 border-b border-slate-800 flex justify-between items-center">
        <div>
          <h2 className="text-lg font-medium">Users</h2>
          <p className="text-sm text-slate-500">Create accounts and manage permissions</p>
        </div>
        <button
          onClick={() => {
            resetForm()
            setEditing(null)
            setShowCreate(true)
          }}
          className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-sm font-medium"
        >
          New user
        </button>
      </header>

      <div className="p-6">
        {isLoading ? (
          <p className="text-slate-500 text-sm">Loading...</p>
        ) : (
          <div className="space-y-3">
            {users.map((u) => (
              <div
                key={u.id}
                className="p-4 bg-slate-900/50 rounded-xl border border-slate-800 flex justify-between gap-4"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium">{u.name}</span>
                    {u.is_admin && (
                      <span className="text-xs px-2 py-0.5 rounded bg-amber-900/50 text-amber-300">
                        Admin
                      </span>
                    )}
                    <span
                      className={`text-xs px-2 py-0.5 rounded ${
                        u.is_active !== false
                          ? 'bg-emerald-900/50 text-emerald-300'
                          : 'bg-slate-700 text-slate-400'
                      }`}
                    >
                      {u.is_active !== false ? 'Active' : 'Inactive'}
                    </span>
                  </div>
                  <p className="text-sm text-slate-500 mt-1">{u.email}</p>
                  {!u.is_admin && (
                    <p className="text-xs text-slate-600 mt-2">
                      {u.permissions.length} permission{u.permissions.length !== 1 ? 's' : ''}
                    </p>
                  )}
                </div>
                <div className="flex flex-col gap-1 shrink-0">
                  <button
                    onClick={() => startEdit(u)}
                    className="text-xs px-2 py-1 bg-slate-700 rounded hover:bg-slate-600"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => {
                      setResetId(u.id)
                      setNewPassword('')
                    }}
                    className="text-xs px-2 py-1 bg-slate-700 rounded hover:bg-slate-600"
                  >
                    Reset password
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {(showCreate || editing) && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center p-4 z-50">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 max-w-lg w-full max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-medium">{editing ? 'Edit user' : 'Create user'}</h3>
            <p className="text-sm text-slate-400 mt-1">
              Set credentials manually and share them with the user offline.
            </p>
            <form
              className="mt-4 space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                if (editing) update.mutate()
                else create.mutate()
              }}
            >
              <label className="block text-sm">
                <span className="text-slate-400">Name</span>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="mt-1 w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2"
                  required
                />
              </label>
              <label className="block text-sm">
                <span className="text-slate-400">Email</span>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="mt-1 w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2"
                  required
                  disabled={!!editing}
                />
              </label>
              {!editing && (
                <label className="block text-sm">
                  <span className="text-slate-400">Password</span>
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="mt-1 w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2"
                    minLength={8}
                    required
                  />
                </label>
              )}
              {editing && (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={isActive}
                    onChange={(e) => setIsActive(e.target.checked)}
                    disabled={editing.id === currentUser?.id}
                  />
                  Active
                  {editing.id === currentUser?.id && (
                    <span className="text-xs text-slate-500">(cannot deactivate yourself)</span>
                  )}
                </label>
              )}
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={isAdminFlag}
                  onChange={(e) => setIsAdminFlag(e.target.checked)}
                  disabled={editing?.id === currentUser?.id && isAdminFlag}
                />
                Administrator (all permissions)
              </label>
              {!isAdminFlag && (
                <PermissionCheckboxes
                  selected={permissions}
                  onChange={setPermissions}
                />
              )}
              {(create.error || update.error) && (
                <p className="text-red-400 text-sm">
                  {((create.error || update.error) as Error).message}
                </p>
              )}
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={create.isPending || update.isPending}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 rounded-lg text-sm font-medium"
                >
                  {editing ? 'Save' : 'Create'}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowCreate(false)
                    setEditing(null)
                    resetForm()
                  }}
                  className="px-4 py-2 bg-slate-700 rounded-lg text-sm"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {resetId && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center p-4 z-50">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 max-w-md w-full">
            <h3 className="text-lg font-medium">Reset password</h3>
            <p className="text-sm text-slate-400 mt-1">Enter a new password to share with the user.</p>
            <form
              className="mt-4 space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                resetPwd.mutate()
              }}
            >
              <label className="block text-sm">
                <span className="text-slate-400">New password</span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="mt-1 w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2"
                  minLength={8}
                  required
                />
              </label>
              {resetPwd.error && (
                <p className="text-red-400 text-sm">{(resetPwd.error as Error).message}</p>
              )}
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={resetPwd.isPending}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 rounded-lg text-sm font-medium"
                >
                  Update password
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setResetId(null)
                    setNewPassword('')
                  }}
                  className="px-4 py-2 bg-slate-700 rounded-lg text-sm"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
