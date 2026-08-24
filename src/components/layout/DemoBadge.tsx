import { FlaskConical } from 'lucide-react'
import { Link } from 'react-router-dom'
import { getDataSource } from '@/services/dataSource'

/**
 * Penanda bahwa layar ini memakai Data Contoh, bukan backend sungguhan.
 *
 * KENAPA INI ADA, dan kenapa ia tidak boleh halus.
 *
 * Dalam mode contoh, `mockInvoiceRepository.send()` tidak menyentuh jaringan
 * sama sekali. Ia mengarang referensi Paper.id, menyusun URL yang terlihat
 * meyakinkan (`pay.paper.id/PPR…`), lalu mencatat "Dikirim ke Paper.id — link
 * pembayaran dibuat" ke jejak audit. Toast suksesnya sama persis dengan
 * pengiriman nyata.
 *
 * Jadi tidak ada satu pun perbedaan yang bisa dilihat antara invoice yang
 * benar-benar terkirim dan yang hanya berpura-pura — sementara penanda sumber
 * data satu-satunya tersembunyi di halaman Pengaturan. Orang bisa menekan
 * "Buat & Kirim ke Paper.id", membaca konfirmasi yang meyakinkan, dan menunggu
 * pembayaran atas invoice yang tidak pernah ada di mana pun.
 *
 * Itu benar-benar terjadi sebelum penanda ini dibuat.
 */
export function DemoBadge() {
  if (getDataSource() !== 'mock') return null

  return (
    <Link
      to="/settings"
      title="Semua data di layar ini contoh. Tidak ada API luar yang dipanggil — termasuk Paper.id. Klik untuk mengganti sumber data."
      className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-amber-300 bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-800 transition-colors hover:bg-amber-100"
    >
      <FlaskConical className="h-3.5 w-3.5" />
      <span className="hidden sm:inline">Data Contoh</span>
      <span className="sm:hidden">Contoh</span>
    </Link>
  )
}
