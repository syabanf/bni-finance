import {
  LayoutGrid,
  FileText,
  Wallet,
  Users,
  Building2,
  Settings,
  RefreshCw,
  type LucideIcon,
} from 'lucide-react'

export interface NavLeaf {
  to: string
  label: string
  /** Exact match for the active state (used for index routes). */
  end?: boolean
}

export type NavNode =
  | { kind: 'section'; label: string }
  | { kind: 'item'; to: string; label: string; icon: LucideIcon; end?: boolean }
  | { kind: 'group'; label: string; icon: LucideIcon; children: NavLeaf[] }

export const NAV: NavNode[] = [
  { kind: 'item', to: '/dashboard', label: 'Dashboard', icon: LayoutGrid },

  { kind: 'section', label: 'Keuangan' },
  {
    kind: 'group',
    label: 'Invoice',
    icon: FileText,
    children: [
      { to: '/invoices', label: 'Semua Invoice', end: true },
      { to: '/invoices/new', label: 'Buat Invoice' },
      { to: '/invoices/renewal-due', label: 'Renewal Due' },
    ],
  },
  { kind: 'item', to: '/payments', label: 'Pembayaran', icon: Wallet },

  { kind: 'section', label: 'Data Member' },
  { kind: 'item', to: '/members', label: 'Member', icon: Users },
  { kind: 'item', to: '/chapters', label: 'Chapter', icon: Building2 },

  { kind: 'section', label: 'Sistem' },
  { kind: 'item', to: '/settings', label: 'Pengaturan Biaya', icon: Settings, end: true },
  { kind: 'item', to: '/settings/sync', label: 'Sinkronisasi Data', icon: RefreshCw },
]
