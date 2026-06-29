import { BrowserRouter, Navigate, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from './auth/AuthContext'
import ProtectedRoute from './auth/ProtectedRoute'
import RequirePerm from './auth/RequirePerm'
import Layout from './Layout'
import SetupPage from './pages/SetupPage'
import LoginPage from './pages/LoginPage'
import InboxPage from './pages/InboxPage'
import ConversationPage from './pages/ConversationPage'
import AccountsPage from './pages/AccountsPage'
import WebhooksPage from './pages/WebhooksPage'
import SettingsPage from './pages/SettingsPage'
import UsersPage from './pages/UsersPage'

const qc = new QueryClient()

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/setup" element={<SetupPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route element={<ProtectedRoute />}>
              <Route element={<Layout />}>
                <Route
                  index
                  element={
                    <RequirePerm perm="view_inbox">
                      <InboxPage />
                    </RequirePerm>
                  }
                />
                <Route
                  path="conversations/:id"
                  element={
                    <RequirePerm perm="view_inbox">
                      <ConversationPage />
                    </RequirePerm>
                  }
                />
                <Route
                  path="accounts"
                  element={
                    <RequirePerm perm="view_accounts">
                      <AccountsPage />
                    </RequirePerm>
                  }
                />
                <Route
                  path="webhooks"
                  element={
                    <RequirePerm perm="view_webhooks">
                      <WebhooksPage />
                    </RequirePerm>
                  }
                />
                <Route
                  path="settings"
                  element={
                    <RequirePerm perm="view_settings">
                      <SettingsPage />
                    </RequirePerm>
                  }
                />
                <Route path="users" element={<UsersPage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Route>
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  )
}
