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
import AdminRedeem from './pages/AdminRedeem'
import AdminUsers from './pages/AdminUsers'
import ModelPricing from './pages/ModelPricing'
import AdminNotice from './pages/AdminNotice'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = useAppStore((s) => s.token)
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const user = useAppStore((s) => s.user)
  if (!user) return <Navigate to="/login" replace />
  if (user.role !== 'admin') return <Navigate to="/dashboard" replace />
  return <>{children}</>
}

const USER_ROUTES = [
  { path: '/dashboard', element: <Dashboard /> },
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
]

export default function App() {
  const { theme } = useAppStore()

  useEffect(() => {
    document.body.setAttribute('theme-mode', theme)
  }, [theme])

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
