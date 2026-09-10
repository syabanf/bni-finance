import { useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { Button, useToast } from '@/components/ui'
import { useCan } from '@/features/auth/usePermission'

/**
 * Tombol tarik-data, dipakai di halaman Member dan Chapter.
 *
 * Dulu keduanya ada di halaman "Sinkronisasi Data" tersendiri. Memindahkannya
 * ke tempat datanya berada menghilangkan satu lompatan yang tidak perlu: orang
 * yang melihat daftar member usang sekarang bisa menyegarkannya di layar yang
 * sama, alih-alih mengingat bahwa ada halaman lain untuk itu.
 *
 * IZINNYA TETAP DIPERIKSA. Halaman lamanya dijaga RequirePermission; kalau
 * tombolnya pindah tanpa membawa penjaganya, aksi yang tadinya terbatas akan
 * muncul untuk semua orang — dan pemindahan yang dimaksudkan kosmetik berubah
 * jadi pelonggaran hak diam-diam.
 */
export function SyncButton({
  jenis,
  onSelesai,
}: {
  jenis: 'member' | 'chapter'
  /** Dipanggil setelah berhasil, supaya halamannya memuat ulang daftarnya. */
  onSelesai?: () => void
}) {
  const { toast } = useToast()
  const boleh = useCan('sync:run')
  const [sibuk, setSibuk] = useState(false)

  if (!boleh) return null

  const jalankan = async () => {
    setSibuk(true)
    try {
      // Impor di dalam handler, bukan di puncak berkas: keduanya menarik
      // seluruh lapisan service, dan halaman Member maupun Chapter tidak perlu
      // membayarnya hanya karena menampilkan sebuah tombol.
      const { chapterService, memberService } = await import('@/services')
      const res = jenis === 'member' ? await memberService.sync() : await chapterService.sync()
      toast(`${res.count} ${jenis} berhasil ditarik dari BNI Visitor Management.`)
      onSelesai?.()
    } catch (err) {
      toast(
        err instanceof Error ? err.message : `Gagal menarik data ${jenis} dari BNI VM.`,
        'error',
      )
    } finally {
      setSibuk(false)
    }
  }

  return (
    <Button variant="outline" onClick={jalankan} loading={sibuk} data-tour="sync-run">
      {!sibuk && <RefreshCw className="h-4 w-4" />}
      Tarik dari BNI VM
    </Button>
  )
}
