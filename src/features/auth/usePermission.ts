import { useAuth } from './AuthContext'
import { can, isAdmin, type Permission } from '@/lib/rbac'

/** Apakah pengguna saat ini boleh melakukan aksi tertentu. */
export function useCan(permission: Permission): boolean {
  const { user } = useAuth()
  return can(user?.role, permission)
}

export function useIsAdmin(): boolean {
  const { user } = useAuth()
  return isAdmin(user?.role)
}
