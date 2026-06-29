import { getToken, setToken, type AuthUser } from './auth/types'

const API = import.meta.env.VITE_API_URL ?? '/api'

export type Account = {
  id: string
  channel: 'rcs' | 'whatsapp'
  name: string
  external_id: string
  client_secret?: string
  access_token?: string
  webhook_url: string
  webhook_verify_token: string
  waba_id: string
  display_phone: string
  sent_delay_ms: number
  delivered_delay_ms: number
  read_delay_ms: number
  failure_rate: number
  auto_read: boolean
}

export type Conversation = {
  id: string
  account_id: string
  channel: string
  user_phone: string
  last_message_at?: string
  unread_count: number
  account_name?: string
  account_external_id?: string
  last_message_preview?: string
}

export type Message = {
  id: string
  conversation_id: string
  direction: 'outbound' | 'inbound'
  vendor_message_id: string
  message_type: string
  status: string
  body: Record<string, unknown>
  created_at: string
}

export type WebhookDelivery = {
  id: string
  channel: string
  event_type: string
  payload: unknown
  http_status?: number
  error_message: string
  created_at: string
}

export type AppUser = AuthUser & {
  is_active?: boolean
  created_at?: string
}

type AuthStatus = {
  needs_setup: boolean
  authenticated: boolean
  user?: AuthUser
}

function authHeaders(): Record<string, string> {
  const token = getToken()
  const h: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) h['Authorization'] = `Bearer ${token}`
  return h
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    ...init,
    headers: { ...authHeaders(), ...init?.headers },
  })
  if (res.status === 401) {
    setToken(null)
    if (!window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/setup')) {
      window.location.href = '/login'
    }
    throw new Error('session expired')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  getApiBase: () => API,
  getToken,

  authStatus: () => request<AuthStatus>('/auth/status'),
  authSetup: (body: { name: string; email: string; password: string }) =>
    request<{ message: string }>('/auth/setup', { method: 'POST', body: JSON.stringify(body) }),
  authLogin: (email: string, password: string) =>
    request<{ token: string; user: AuthUser }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  authLogout: () => request<{ message: string }>('/auth/logout', { method: 'POST' }),
  authMe: () => request<AuthUser>('/auth/me'),
  listPermissions: () => request<string[]>('/permissions'),

  listUsers: () => request<AppUser[]>('/users'),
  createUser: (body: {
    name: string
    email: string
    password: string
    permissions: string[]
    is_admin?: boolean
  }) => request<AppUser>('/users', { method: 'POST', body: JSON.stringify(body) }),
  updateUser: (
    id: string,
    body: { name: string; is_active: boolean; permissions: string[]; is_admin?: boolean },
  ) => request<AppUser>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  resetUserPassword: (id: string, password: string) =>
    request<{ message: string }>(`/users/${id}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password }),
    }),

  health: () => request<{ status: string }>('/health'),
  listAccounts: () => request<Account[]>('/accounts'),
  createAccount: (body: Partial<Account>) =>
    request<Account>('/accounts', { method: 'POST', body: JSON.stringify(body) }),
  updateAccount: (id: string, body: Partial<Account>) =>
    request<Account>(`/accounts/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteAccount: (id: string) => request<void>(`/accounts/${id}`, { method: 'DELETE' }),
  listConversations: () => request<Conversation[]>('/conversations'),
  getConversation: (id: string) => request<Conversation>(`/conversations/${id}`),
  listMessages: (id: string) => request<Message[]>(`/conversations/${id}/messages`),
  sendInbound: (id: string, text: string) =>
    request<Message>(`/conversations/${id}/messages`, {
      method: 'POST',
      body: JSON.stringify({ text }),
    }),
  markRead: (id: string) => request<void>(`/conversations/${id}/read`, { method: 'POST' }),
  triggerStatus: (messageId: string, status: string) =>
    request<void>(`/messages/${messageId}/status`, {
      method: 'POST',
      body: JSON.stringify({ status }),
    }),
  listWebhooks: (limit = 50) => request<WebhookDelivery[]>(`/webhooks?limit=${limit}`),
  purgeData: (scope: 'messages' | 'all') =>
    request<{ status: string; scope: string }>(`/data?scope=${scope}`, { method: 'DELETE' }),
}

export function messagePreview(body: Record<string, unknown>): string {
  if (typeof body.plainText === 'string') return body.plainText
  const content = body.content as Record<string, unknown> | undefined
  if (content && typeof content.plainText === 'string') return content.plainText
  const text = body.text as Record<string, unknown> | undefined
  if (text && typeof text.body === 'string') return text.body
  if (body.richCardDetails) return '[Rich Card]'
  if (body.carousel) return '[Carousel]'
  if (body.type === 'template') return '[Template]'
  return JSON.stringify(body).slice(0, 80)
}
