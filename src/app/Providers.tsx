import { Outlet } from 'react-router-dom'
import { AuthProvider } from '@/features/auth/AuthContext'
import { ToastProvider } from '@/components/ui'

/** Top-level providers wrapping every route (public + protected). */
export function Providers() {
  return (
    <AuthProvider>
      <ToastProvider>
        <Outlet />
      </ToastProvider>
    </AuthProvider>
  )
}
