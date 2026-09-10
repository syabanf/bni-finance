import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Bell, ChevronDown, LogOut, Search, UserCircle2 } from 'lucide-react'
import { Avatar, BniLogo } from '@/components/ui'
import { useAuth } from '@/features/auth/AuthContext'
import { useNotifications } from '@/features/notifications/NotificationsContext'
import { cn } from '@/lib/cn'
import { TourButton } from '@/features/tour/TourButton'
import { DemoBadge } from './DemoBadge'
import { ROLE_LABEL } from '@/lib/rbac'

/**
 * Tujuan pencarian topbar.
 *
 * Setiap rute di sini HARUS membaca `?q=` dan mengisi kotak pencariannya
 * sendiri dari sana. Menambahkan tujuan yang tidak melakukannya menghasilkan
 * halaman yang terbuka dengan kotak kosong — orangnya sudah mengetik, dan yang
 * terlihat tetap seluruh daftar.
 */
const TUJUAN = [
  { nilai: 'invoices', label: 'Invoice', rute: '/invoices', petunjuk: 'Nomor invoice atau nama member…' },
  { nilai: 'members', label: 'Member', rute: '/members', petunjuk: 'Nama, ID, atau email…' },
  { nilai: 'payments', label: 'Pembayaran', rute: '/payments', petunjuk: 'Nama member atau nomor invoice…' },
  { nilai: 'chapters', label: 'Chapter', rute: '/chapters', petunjuk: 'Nama chapter, kota, atau area…' },
] as const

type Entitas = (typeof TUJUAN)[number]['nilai']

const KUNCI_ENTITAS = 'bni.cari.entitas'

function entitasTersimpan(): Entitas {
  try {
    const v = localStorage.getItem(KUNCI_ENTITAS)
    if (TUJUAN.some((t) => t.nilai === v)) return v as Entitas
  } catch {
    // Peramban yang memblokir penyimpanan situs melempar di sini. Pilihannya
    // kembali ke bawaan; tidak ada yang perlu gagal karena itu.
  }
  return 'invoices'
}

export function Topbar() {
  const { user, logout } = useAuth()
  const { unreadCount } = useNotifications()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [entitas, setEntitas] = useState<Entitas>(entitasTersimpan)
  const menuRef = useRef<HTMLDivElement>(null)

  const simpanEntitas = (v: Entitas) => {
    setEntitas(v)
    try {
      localStorage.setItem(KUNCI_ENTITAS, v)
    } catch {
      // Tidak bisa disimpan bukan alasan untuk tidak bisa dipakai.
    }
  }

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <header className="sticky top-0 z-30 border-b border-ink-100 bg-white/90 backdrop-blur-md safe-top">
      <div className="flex h-16 items-center gap-3 px-4 lg:px-6">
        {/* Mobile brand (sidebar logo is hidden on mobile) */}
        <Link to="/dashboard" className="flex items-center gap-2.5 lg:hidden">
          <BniLogo className="h-7 w-auto" />
          <span className="border-l border-ink-100 pl-2.5 text-[15px] font-bold text-ink-900">Finance Hub</span>
        </Link>

        <DemoBadge />

        {/*
          Pencarian: TUJUANNYA DIPILIH, tidak lagi selalu ke daftar invoice.
          
          Sebelumnya apa pun yang diketik berujung ke /invoices?q= — termasuk
          nama member yang belum punya invoice, yang karena itu tidak akan
          pernah ketemu, padahal placeholder-nya menjanjikan "Cari invoice,
          member…". Yang rusak bukan pencariannya melainkan janjinya.
          
          Pilihannya disimpan di localStorage: orang yang bekerja di data
          member mencari member berkali-kali berturut-turut, dan mengembalikan
          pilihan ke "Invoice" setiap kali halaman dimuat ulang berarti
          menyuruhnya memilih lagi sepanjang hari.
        */}
        <form
          onSubmit={(e) => {
            e.preventDefault()
            const q = search.trim()
            const tujuan = TUJUAN.find((t) => t.nilai === entitas) ?? TUJUAN[0]
            navigate(q ? `${tujuan.rute}?q=${encodeURIComponent(q)}` : tujuan.rute)
          }}
          className="relative ml-auto hidden w-full max-w-sm sm:block"
          data-tour="topbar-search"
        >
          <div className="flex h-9 items-center rounded-xl border border-ink-200 bg-ink-50 transition-colors focus-within:border-brand-400 focus-within:bg-white">
            <select
              value={entitas}
              onChange={(e) => simpanEntitas(e.target.value as Entitas)}
              aria-label="Cari di"
              className="h-full shrink-0 rounded-l-xl border-0 bg-transparent pl-3 pr-1 text-xs font-medium text-ink-600 focus-ring"
            >
              {TUJUAN.map((t) => (
                <option key={t.nilai} value={t.nilai}>
                  {t.label}
                </option>
              ))}
            </select>
            <span className="h-4 w-px shrink-0 bg-ink-200" aria-hidden />
            {/*
              Ikon kaca pembesar ini TOMBOL SUBMIT, bukan hiasan.
              
              Enter tetap bekerja tanpanya — yang memblokir implicit submission
              hanyalah adanya LEBIH DARI SATU field bertipe teks, dan <select>
              tidak termasuk. Tombol ini ada karena alasan lain: pemilih di
              sebelah kiri membuat kotaknya terlihat seperti kontrol majemuk,
              dan kontrol majemuk tanpa satu pun tombol menyisakan pertanyaan
              "lalu ditekan apa". Ia juga memberi sasaran sentuh untuk yang
              memakai layar sentuh, di mana tidak ada Enter untuk ditekan.
            */}
            <button
              type="submit"
              aria-label="Cari"
              className="ml-1 shrink-0 rounded-lg p-1 text-ink-400 transition-colors hover:text-ink-700 focus-ring"
            >
              <Search className="h-4 w-4" />
            </button>
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={(TUJUAN.find((t) => t.nilai === entitas) ?? TUJUAN[0]).petunjuk}
              aria-label={`Cari ${(TUJUAN.find((t) => t.nilai === entitas) ?? TUJUAN[0]).label}`}
              className="h-full w-full bg-transparent px-1.5 text-sm text-ink-700 placeholder:text-ink-400 focus:outline-none"
            />
          </div>
        </form>

        {/* Notifications */}
        {/* ml-auto pindah ke sini: di ponsel pencarian disembunyikan, jadi
            kelompok tombol inilah yang harus terdorong ke kanan. */}
        <div className="ml-auto flex items-center gap-1 sm:ml-0">
          <TourButton />
        </div>

        <Link
          to="/notifications"
          data-tour="topbar-notifications"
          className="relative rounded-xl p-2 text-ink-500 transition-colors hover:bg-ink-100"
          aria-label={`Notifikasi${unreadCount > 0 ? ` (${unreadCount} belum dibaca)` : ''}`}
        >
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-brand-500 px-1 text-[10px] font-bold leading-none text-white ring-2 ring-white">
              {unreadCount > 9 ? '9+' : unreadCount}
            </span>
          )}
        </Link>

        {/* User menu */}
        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setMenuOpen((o) => !o)}
            className="flex items-center gap-2.5 rounded-xl py-1.5 pl-2 pr-2.5 transition-colors hover:bg-ink-100"
          >
            <div className="hidden text-right leading-tight sm:block">
              <div className="text-sm font-semibold text-ink-900">{user?.name ?? 'Admin'}</div>
              <div className="text-xs text-ink-400">
                {user?.role ? ROLE_LABEL[user.role] : '—'}
              </div>
            </div>
            <Avatar name={user?.name ?? 'Admin'} size="sm" />
            <ChevronDown className={cn('h-4 w-4 text-ink-400 transition-transform', menuOpen && 'rotate-180')} />
          </button>

          {menuOpen && (
            <div className="absolute right-0 top-full mt-2 w-56 overflow-hidden rounded-xl border border-ink-100 bg-white py-1.5 shadow-card-hover animate-fade-in">
              <div className="border-b border-ink-100 px-4 py-3">
                <div className="text-sm font-semibold text-ink-900">{user?.name}</div>
                <div className="truncate text-xs text-ink-400">{user?.email}</div>
              </div>
              <Link
                to="/profile"
                onClick={() => setMenuOpen(false)}
                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-ink-600 hover:bg-ink-50"
              >
                <UserCircle2 className="h-4 w-4" />
                Profil Saya
              </Link>
              <button
                onClick={handleLogout}
                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-red-600 hover:bg-red-50"
              >
                <LogOut className="h-4 w-4" />
                Keluar
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
