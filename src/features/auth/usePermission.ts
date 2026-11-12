import { useAuth } from './AuthContext'
import { can, type Permission } from '@/lib/rbac'

/** Apakah pengguna saat ini boleh melakukan aksi tertentu. */
export function useCan(permission: Permission): boolean {
  const { user } = useAuth()
  return can(user?.role, permission)
}

