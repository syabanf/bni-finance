import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { ShieldAlert } from 'lucide-react'
import { Button, Card, CardBody } from '@/components/ui'
import { useAuth } from './AuthContext'
import { can, ROLE_LABEL, type Permission } from '@/lib/rbac'

/**
 * Menjaga rute berdasarkan izin. Menampilkan pesan "tanpa akses" alih-alih
 * diam-diam redirect, supaya jelas kenapa halaman tidak terbuka.
 *
 * Ini pembatas UI — batas sebenarnya tetap RLS di database.
 */
export function RequirePermission({
  permission,
  children,
}: {
  permission: Permission
  children: ReactNode
}) {
  const { user } = useAuth()
  const navigate = useNavigate()

  if (can(user?.role, permission)) return <>{children}</>

  return (
    <div className="flex min-h-[60vh] items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardBody className="flex flex-col items-center py-10 text-center">
          <span className="flex h-14 w-14 items-center justify-center rounded-full bg-amber-50 text-amber-500">
            <ShieldAlert className="h-7 w-7" />
          </span>
          <h1 className="mt-4 text-xl font-bold text-ink-900">Tidak punya akses</h1>
          <p className="mt-1.5 max-w-sm text-sm text-ink-500">
            Halaman ini khusus Administrator. Peran Anda saat ini:{' '}
            <span className="font-medium text-ink-700">
              {user?.role ? ROLE_LABEL[user.role] : '—'}
            </span>
            .
          </p>
          <Button className="mt-6" variant="outline" onClick={() => navigate('/dashboard')}>
            Kembali ke Dashboard
          </Button>
        </CardBody>
      </Card>
    </div>
  )
}
