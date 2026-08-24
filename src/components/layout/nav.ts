import {
  LayoutGrid,
  FileText,
  Wallet,
  Users,
  Building2,
  CalendarCheck,
  Settings,
  AlertTriangle,
  BarChart3,
  TerminalSquare,
  type LucideIcon,
} from 'lucide-react'
import type { Permission } from '@/lib/rbac'

export interface NavLeaf {
  to: string
  label: string
  /** Exact match for the active state (used for index routes). */
  end?: boolean
  /** Bila diisi, leaf hanya tampil untuk peran yang punya izin ini. */
  permission?: Permission
}

export type NavNode =
  | { kind: 'section'; label: string; permission?: Permission }
  | {
      kind: 'item'
      to: string
      label: string
      icon: LucideIcon
      end?: boolean
      urgent?: boolean
      /** Bila diisi, item hanya tampil untuk peran yang punya izin ini. */
      permission?: Permission
    }
  | {
      kind: 'group'
      label: string
      icon: LucideIcon
      children: NavLeaf[]
      permission?: Permission
    }

export const NAV: NavNode[] = [
  { kind: 'item', to: '/dashboard', label: 'Dashboard', icon: LayoutGrid },
  { kind: 'item', to: '/urgent', label: 'Perlu Tindakan', icon: AlertTriangle, urgent: true },

  { kind: 'section', label: 'Keuangan' },
  { kind: 'item', to: '/invoices', label: 'Semua Invoice', icon: FileText },
  { kind: 'item', to: '/payments', label: 'Pembayaran', icon: Wallet },
  { kind: 'item', to: '/reports', label: 'Laporan', icon: BarChart3 },

  { kind: 'section', label: 'Data Member' },
  { kind: 'item', to: '/members', label: 'Member', icon: Users },
  { kind: 'item', to: '/chapters', label: 'Chapter', icon: Building2 },
  { kind: 'item', to: '/renewal', label: 'Konfirmasi Renewal', icon: CalendarCheck },

  // Sistem dikelompokkan agar sidebar tidak memanjang: menu harian di atas,
  // urusan konfigurasi dan alat teknis terlipat sampai dibutuhkan.
  { kind: 'section', label: 'Sistem', permission: 'settings:manage' },
  {
    kind: 'group',
    label: 'Pengaturan',
    icon: Settings,
    permission: 'settings:manage',
    children: [
      { to: '/settings', label: 'Biaya Keanggotaan', end: true },
      { to: '/settings/sync', label: 'Sinkronisasi Data', permission: 'sync:run' },
      { to: '/settings/users', label: 'Pengguna' },
      { to: '/settings/import', label: 'Impor Data' },
    ],
  },
  {
    kind: 'group',
    label: 'Alat Teknis',
    icon: TerminalSquare,
    permission: 'settings:manage',
    children: [
      { to: '/api-console', label: 'Konsol API' },
      { to: '/blackbox', label: 'Blackbox Integrasi' },
    ],
  },
]
