import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useSSE } from './hooks/useSSE'
import { useAuth } from './auth/AuthContext'

const navItems = [
  { to: '/', label: 'Inbox', perm: 'view_inbox' },
  { to: '/accounts', label: 'Accounts', perm: 'view_accounts' },
  { to: '/webhooks', label: 'Webhooks', perm: 'view_webhooks' },
  { to: '/settings', label: 'Settings', perm: 'view_settings' },
  { to: '/users', label: 'Users', adminOnly: true },
]

export default function Layout() {
  useSSE()
  const loc = useLocation()
  const navigate = useNavigate()
  const { user, logout, can, isAdmin } = useAuth()

  const visibleNav = navItems.filter((item) => {
    if (item.adminOnly) return isAdmin
    return can(item.perm!)
  })

  return (
    <div className="flex h-screen">
      <aside className="w-56 shrink-0 border-r border-slate-800 bg-slate-900 flex flex-col">
        <div className="p-4 border-b border-slate-800">
          <h1 className="font-semibold text-sm tracking-wide text-emerald-400">Vendor Simulator</h1>
          <p className="text-xs text-slate-500 mt-1">WhatsApp + RCS</p>
        </div>
        <nav className="flex-1 p-2 space-y-1">
          {visibleNav.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className={`block px-3 py-2 rounded-lg text-sm transition-colors ${
                loc.pathname === item.to || (item.to !== '/' && loc.pathname.startsWith(item.to))
                  ? 'bg-slate-800 text-white'
                  : 'text-slate-400 hover:bg-slate-800/50 hover:text-white'
              }`}
            >
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="p-4 border-t border-slate-800">
          <p className="text-xs text-slate-400 truncate">{user?.name}</p>
          <p className="text-xs text-slate-600 truncate">{user?.email}</p>
          <button
            onClick={() => {
              logout()
              navigate('/login')
            }}
            className="mt-2 text-xs text-slate-400 hover:text-white"
          >
            Sign out
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-hidden">
        <Outlet />
      </main>
    </div>
  )
}
