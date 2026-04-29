import React, { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAppStore } from './store'
import AppLayout from './components/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import Dashboard from './pages/Dashboard'
import Tokens from './pages/Tokens'
import Models from './pages/Models'
import Logs from './pages/Logs'
import Wallet from './pages/Wallet'
import Channels from './pages/Channels'
import Settings from './pages/Settings'
import Branding from './pages/Branding'
import AdminRedeem from './pages/AdminRedeem'
import AdminUsers from './pages/AdminUsers'
import ModelPricing from './pages/ModelPricing'
import AdminNotice from './pages/AdminNotice'
import AdminAudit from './pages/AdminAudit'
import StatusPage from './pages/Status'
import ChatEmbed from './pages/ChatEmbed'
import { getBranding } from './api'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const loggedIn = useAppStore((s) => s.loggedIn)
  if (!loggedIn) return <Navigate to="/login" replace />
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const user = useAppStore((s) => s.user)
  if (!user) return <Navigate to="/login" replace />
  if (user.role !== 'admin') return <Navigate to="/dashboard" replace />
  return <>{children}</>
}

const USER_ROUTES = [
  { path: '/chat-embed', element: <ChatEmbed /> },
  { path: '/dashboard', element: <Dashboard /> },
  { path: '/status', element: <StatusPage /> },
  { path: '/tokens', element: <Tokens /> },
  { path: '/models', element: <Models /> },
  { path: '/logs', element: <Logs /> },
  { path: '/wallet', element: <Wallet /> },
  { path: '/settings', element: <Settings /> },
]

const ADMIN_ROUTES = [
  { path: '/channels', element: <Channels /> },
  { path: '/admin/pricing', element: <ModelPricing /> },
  { path: '/admin/notice', element: <AdminNotice /> },
  { path: '/admin/redeem', element: <AdminRedeem /> },
  { path: '/admin/users', element: <AdminUsers /> },
  { path: '/admin/branding', element: <Branding /> },
  { path: '/admin/audit', element: <AdminAudit /> },
]

export default function App() {
  const { theme } = useAppStore()

  useEffect(() => {
    document.body.setAttribute('theme-mode', theme)
  }, [theme])

  useEffect(() => {
    getBranding()
      .then((res) => {
        const d = res.data || {}
        if (d.site_title) document.title = d.site_title
        if (d.site_favicon) setFavicon(d.site_favicon)
      })
      .catch(() => {})
  }, [])

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />

        <Route
          path="/"
          element={
            <PrivateRoute>
              <AppLayout>
                <Navigate to="/dashboard" replace />
              </AppLayout>
            </PrivateRoute>
          }
        />

        {USER_ROUTES.map(({ path, element }) => (
          <Route
            key={path}
            path={path}
            element={
              <PrivateRoute>
                <AppLayout>{element}</AppLayout>
              </PrivateRoute>
            }
          />
        ))}

        {ADMIN_ROUTES.map(({ path, element }) => (
          <Route
            key={path}
            path={path}
            element={
              <AdminRoute>
                <AppLayout>{element}</AppLayout>
              </AdminRoute>
            }
          />
        ))}

        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

function setFavicon(value: string) {
  const old = document.querySelector('link[rel="icon"]')
  if (old) old.remove()

  const link = document.createElement('link')
  link.rel = 'icon'

  // Detect emoji (short non-URL value) vs image URL
  const isEmoji = !value.includes('.') && !value.includes('/') && [...value].length <= 4
  if (isEmoji) {
    link.href = `data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">${value}</text></svg>`
    link.type = 'image/svg+xml'
  } else {
    link.href = value
  }

  document.head.appendChild(link)
}
