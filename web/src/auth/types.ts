export const PERMISSIONS = {
  view_inbox: 'Inbox',
  view_accounts: 'Accounts',
  view_webhooks: 'Webhooks',
  view_settings: 'Settings',
  view_users: 'Users',
  action_reply: 'Reply in conversations',
  action_delivery: 'Trigger delivery status',
  action_accounts_write: 'Manage vendor accounts',
  action_data_purge: 'Clear all data',
  action_users_manage: 'Manage users',
} as const

export type Permission = keyof typeof PERMISSIONS

export type AuthUser = {
  id: string
  email: string
  name: string
  is_admin: boolean
  permissions: string[]
}

const TOKEN_KEY = 'sim_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export function can(user: AuthUser | null, perm: string): boolean {
  if (!user) return false
  if (user.is_admin) return true
  return user.permissions.includes(perm)
}
