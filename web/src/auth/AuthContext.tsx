import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import { can, getToken, setToken, type AuthUser } from './types'

type AuthState = {
  user: AuthUser | null
  loading: boolean
  needsSetup: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  refresh: () => Promise<void>
  can: (perm: string) => boolean
  isAdmin: boolean
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)
  const [needsSetup, setNeedsSetup] = useState(false)

  const refresh = useCallback(async () => {
    const status = await api.authStatus()
    setNeedsSetup(status.needs_setup)
    if (status.needs_setup) {
      setUser(null)
      setToken(null)
      setLoading(false)
      return
    }
    const token = getToken()
    if (!token) {
      setUser(null)
      setLoading(false)
      return
    }
    try {
      const me = await api.authMe()
      setUser(me)
    } catch {
      setUser(null)
      setToken(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.authLogin(email, password)
    setToken(res.token)
    setUser(res.user)
    setNeedsSetup(false)
  }, [])

  const logout = useCallback(() => {
    setToken(null)
    setUser(null)
    api.authLogout().catch(() => {})
  }, [])

  const value = useMemo(
    () => ({
      user,
      loading,
      needsSetup,
      login,
      logout,
      refresh,
      can: (perm: string) => can(user, perm),
      isAdmin: user?.is_admin ?? false,
    }),
    [user, loading, needsSetup, login, logout, refresh],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
