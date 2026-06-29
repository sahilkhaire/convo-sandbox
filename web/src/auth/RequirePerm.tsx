import { Navigate } from 'react-router-dom'
import { useAuth } from './AuthContext'

export default function RequirePerm({ perm, children }: { perm: string; children: React.ReactNode }) {
  const { can, loading } = useAuth()
  if (loading) return null
  if (!can(perm)) return <Navigate to="/" replace />
  return <>{children}</>
}
