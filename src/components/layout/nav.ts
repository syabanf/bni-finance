import {
  LayoutGrid,
  FileText,
  Wallet,
  Users,
  Building2,
  CalendarCheck,
  CalendarClock,
  Settings,
  AlertTriangle,
  BarChart3,
  TerminalSquare,
  BookOpen,
  FileCheck2,
  Globe2,
  type LucideIcon,
} from 'lucide-react'
import { JUDUL_INTEGRASI } from '@/lib/fitur'
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
  { kind: 'item', to: '/payments/proofs', label: 'Bukti Bayar', icon: FileCheck2 },
  { kind: 'item', to: '/reports', label: 'Laporan', icon: BarChart3 },

  { kind: 'section', label: 'Data Member' },
  { kind: 'item', to: '/members', label: 'Member', icon: Users },
  { kind: 'item', to: '/members/lama-keanggotaan', label: 'Lama Keanggotaan', icon: CalendarClock },
  { kind: 'item', to: '/chapters', label: 'Chapter', icon: Building2 },
  { kind: 'item', to: '/renewal', label: 'Konfirmasi Renewal', icon: CalendarCheck },
  {
    kind: 'item',
    to: '/renewal/submissions',
    label: 'Pengajuan BNI Global',
    icon: Globe2,
    permission: 'invoice:manage',
  },

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
      ...(JUDUL_INTEGRASI ? [{ to: '/integrasi', label: JUDUL_INTEGRASI }] : []),
      { to: '/api-console', label: 'Konsol API' },
      { to: '/blackbox', label: 'Blackbox Integrasi' },
    ],
  },

  { kind: 'section', label: 'Bantuan' },
  { kind: 'item', to: '/panduan', label: 'Panduan', icon: BookOpen },
]

const SEMUA_TUJUAN = NAV.flatMap((n): { to: string; end?: boolean }[] =>
  n.kind === 'item' ? [n] : n.kind === 'group' ? n.children : [],
)

/**
 * Tujuan menu yang cocok dengan alamat ini. Bila beberapa cocok, yang
 * terpanjang menang.
 *
 * `/members/lama-keanggotaan` berawalan `/members`, jadi dengan pencocokan
 * awalan biasa menu Member ikut menyala di halaman Lama Keanggotaan dan dua
 * menu tampak terpilih sekaligus. Halaman detail seperti `/members/123` tetap
 * menyalakan Member, karena tidak ada tujuan lain yang lebih panjang.
 */
export function tujuanAktif(pathname: string): string | null {
  let terbaik: string | null = null
  for (const { to, end } of SEMUA_TUJUAN) {
    const cocok = pathname === to || (!end && pathname.startsWith(`${to}/`))
    if (cocok && (terbaik === null || to.length > terbaik.length)) terbaik = to
  }
  return terbaik
}
