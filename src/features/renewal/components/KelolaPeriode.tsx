import { useEffect, useState } from 'react'
import { CalendarPlus, Pencil, Trash2, Wand2 } from 'lucide-react'
import type { RenewalPeriod } from '@/types'
import { Badge, Button, Field, Input, Modal, useToast } from '@/components/ui'
import { periodService } from '@/services'
import { formatDate } from '@/lib/format'
import { hariIni, namaDariRentang, periodeBeririsan, rentangBerikutnya } from '@/lib/periode'
import { cn } from '@/lib/cn'

interface Formulir {
  name: string
  startDate: string
  endDate: string
  /** true selama admin belum mengetik nama sendiri: nama mengikuti tanggal. */
  namaOtomatis: boolean
}

function formulirBaru(periode: RenewalPeriod[]): Formulir {
  const { mulai, selesai } = rentangBerikutnya(periode)
  return { name: namaDariRentang(mulai, selesai), startDate: mulai, endDate: selesai, namaOtomatis: true }
}

function statusPeriode(p: RenewalPeriod, hari: string) {
  if (hari < p.startDate) return { label: 'Mendatang', tone: 'blue' as const }
  if (hari > p.endDate) return { label: 'Selesai', tone: 'gray' as const }
  return { label: 'Berjalan', tone: 'green' as const }
}

/**
 * Master data periode konfirmasi renewal, untuk admin.
 *
 * Formulirnya langsung terisi periode sesudah periode terakhir, jadi menambah
 * "2029 · Jul–Des" cukup satu klik. Nama mengikuti tanggal sampai admin
 * mengetik nama sendiri. Bentrok nama dan irisan rentang ditampilkan sebelum
 * Simpan ditekan; server tetap memeriksanya lagi.
 */
export function KelolaPeriode({
  open,
  onClose,
  periode,
  onBerubah,
}: {
  open: boolean
  onClose: () => void
  periode: RenewalPeriod[]
  /** Dipanggil sesudah tiap perubahan. `pilih` adalah periode yang baru dibuat. */
  onBerubah: (pilih?: string) => void
}) {
  const { toast } = useToast()
  const [form, setForm] = useState<Formulir>(() => formulirBaru(periode))
  const [diubah, setDiubah] = useState<RenewalPeriod | null>(null)
  const [menyimpan, setMenyimpan] = useState(false)
  const [akanDihapus, setAkanDihapus] = useState<string | null>(null)
  const [menghapus, setMenghapus] = useState(false)
  const hari = hariIni()

  // Dibuka ulang dalam mode tambah, dengan saran dari daftar terbaru.
  useEffect(() => {
    if (!open) return
    setDiubah(null)
    setAkanDihapus(null)
    setForm(formulirBaru(periode))
    // Hanya saat dibuka: daftar yang berubah sesudah Simpan tidak boleh
    // menimpa formulir yang sedang diisi.
  }, [open])

  const ubahTanggal = (bagian: Partial<Pick<Formulir, 'startDate' | 'endDate'>>) =>
    setForm((f) => {
      const baru = { ...f, ...bagian }
      return baru.namaOtomatis ? { ...baru, name: namaDariRentang(baru.startDate, baru.endDate) } : baru
    })

  const nama = form.name.trim()
  const terbalik = !!form.startDate && !!form.endDate && form.endDate < form.startDate
  // Saat mengubah, irisan hanya diperiksa bila tanggalnya bergeser, sama
  // seperti server: periode lama berbentuk tahun beririsan dengan semesternya.
  const tanggalBergeser = !diubah || diubah.startDate !== form.startDate || diubah.endDate !== form.endDate
  const irisan =
    !terbalik && tanggalBergeser && form.startDate && form.endDate
      ? periodeBeririsan(periode, form.startDate, form.endDate, diubah?.id)
      : undefined
  const namaGanda = periode.find((p) => p.id !== diubah?.id && p.name.toLowerCase() === nama.toLowerCase())
  const bolehSimpan = !!nama && !!form.startDate && !!form.endDate && !terbalik && !irisan && !namaGanda

  const mulaiUbah = (p: RenewalPeriod) => {
    setAkanDihapus(null)
    setDiubah(p)
    setForm({ name: p.name, startDate: p.startDate, endDate: p.endDate, namaOtomatis: false })
  }

  const batalUbah = () => {
    setDiubah(null)
    setForm(formulirBaru(periode))
  }

  const simpan = async () => {
    if (!bolehSimpan) return
    setMenyimpan(true)
    try {
      const isi = { name: nama, startDate: form.startDate, endDate: form.endDate }
      if (diubah) {
        await periodService.update(diubah.id, isi)
        toast(`Periode ${nama} diperbarui.`)
        setDiubah(null)
        onBerubah()
      } else {
        const baru = await periodService.create(isi)
        toast(`Periode ${baru.name} ditambahkan.`)
        onBerubah(baru.id)
      }
      // Saran berikutnya menyambung periode yang baru disimpan.
      const sesudah = diubah
        ? periode.map((p) => (p.id === diubah.id ? { ...p, ...isi } : p))
        : [...periode, { id: '', requestCount: 0, ...isi }]
      setForm(formulirBaru(sesudah))
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan periode.', 'error')
    } finally {
      setMenyimpan(false)
    }
  }

  const hapus = async (p: RenewalPeriod) => {
    setMenghapus(true)
    try {
      await periodService.remove(p.id)
      toast(`Periode ${p.name} dihapus.`)
      setAkanDihapus(null)
      if (diubah?.id === p.id) batalUbah()
      onBerubah()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menghapus periode.', 'error')
    } finally {
      setMenghapus(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="lg"
      title="Kelola Periode"
      description="Periode yang bisa dipilih di Konfirmasi Renewal. Berlaku untuk semua chapter."
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          simpan()
        }}
        className={cn('rounded-xl border p-4', diubah ? 'border-amber-200 bg-amber-50/40' : 'border-ink-200 bg-white/60')}
      >
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold text-ink-900">
            {diubah ? `Ubah periode ${diubah.name}` : 'Tambah periode'}
          </h3>
          {!diubah && (
            <button
              type="button"
              onClick={() => setForm(formulirBaru(periode))}
              className="inline-flex items-center gap-1 text-xs font-medium text-brand-600 hover:text-brand-700"
            >
              <Wand2 className="h-3.5 w-3.5" />
              Sambung periode terakhir
            </button>
          )}
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Tanggal mulai" required>
            <Input
              type="date"
              value={form.startDate}
              onChange={(e) => ubahTanggal({ startDate: e.target.value })}
              data-testid="periode-mulai"
            />
          </Field>
          <Field label="Tanggal selesai" required error={terbalik ? 'Tanggal selesai sebelum tanggal mulai.' : undefined}>
            <Input
              type="date"
              value={form.endDate}
              min={form.startDate || undefined}
              onChange={(e) => ubahTanggal({ endDate: e.target.value })}
              data-testid="periode-selesai"
            />
          </Field>
          <Field
            label="Nama periode"
            required
            className="sm:col-span-2"
            error={namaGanda ? `Nama ini sudah dipakai periode ${namaGanda.name}.` : undefined}
            hint={
              form.namaOtomatis ? (
                'Mengikuti tanggal. Ketik untuk memakai nama sendiri.'
              ) : (
                <button
                  type="button"
                  className="text-brand-600 hover:text-brand-700"
                  onClick={() =>
                    setForm((f) => ({ ...f, name: namaDariRentang(f.startDate, f.endDate), namaOtomatis: true }))
                  }
                >
                  Pakai nama dari tanggal
                </button>
              )
            }
          >
            <Input
              value={form.name}
              maxLength={60}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value, namaOtomatis: false }))}
              placeholder="2029 · Jul–Des"
              data-testid="periode-nama"
            />
          </Field>
        </div>

        {irisan && (
          <p className="mt-3 rounded-lg bg-red-50 px-3 py-2 text-xs leading-relaxed text-red-700" data-testid="periode-irisan">
            Rentang ini beririsan dengan {irisan.name} ({formatDate(irisan.startDate)} sampai{' '}
            {formatDate(irisan.endDate)}). Member yang jatuh tempo di irisannya bisa ditanyakan dua kali.
          </p>
        )}
        {diubah && diubah.requestCount > 0 && tanggalBergeser && !irisan && (
          <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs leading-relaxed text-amber-800">
            Periode ini sudah punya {diubah.requestCount} permintaan konfirmasi. Permintaannya tetap di periode
            ini, tetapi daftar jatuh tempo saat meminta konfirmasi berikutnya mengikuti tanggal yang baru.
          </p>
        )}

        <div className="mt-4 flex flex-wrap justify-end gap-2">
          {diubah && (
            <Button type="button" variant="ghost" onClick={batalUbah}>
              Batal
            </Button>
          )}
          <Button type="submit" loading={menyimpan} disabled={!bolehSimpan} data-testid="periode-simpan">
            {diubah ? (
              'Simpan perubahan'
            ) : (
              <>
                <CalendarPlus className="h-4 w-4" />
                Tambah periode
              </>
            )}
          </Button>
        </div>
      </form>

      <div className="mt-5">
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-ink-900">Daftar periode</h3>
          <span className="text-xs text-ink-400">{periode.length} periode</span>
        </div>
        {periode.length === 0 ? (
          <p className="rounded-xl bg-ink-50 px-4 py-6 text-center text-sm text-ink-500">
            Belum ada periode. Tambahkan yang pertama lewat formulir di atas.
          </p>
        ) : (
          <ul className="divide-y divide-ink-100 rounded-xl border border-ink-100 bg-white/60" data-testid="daftar-periode">
            {periode.map((p) => {
              const st = statusPeriode(p, hari)
              const dipakai = p.requestCount > 0
              return (
                <li
                  key={p.id}
                  className={cn(
                    'flex flex-wrap items-center gap-x-3 gap-y-2 px-4 py-3',
                    diubah?.id === p.id && 'bg-amber-50/60',
                  )}
                >
                  {/* Lebar minimum membuat tombol aksi turun ke baris sendiri
                      di layar sempit, alih-alih menghimpit nama periodenya. */}
                  <div className="min-w-[13rem] flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-ink-900">{p.name}</span>
                      <Badge tone={st.tone}>{st.label}</Badge>
                    </div>
                    <div className="mt-0.5 text-xs text-ink-500">
                      {formatDate(p.startDate)} sampai {formatDate(p.endDate)} ·{' '}
                      {dipakai ? `${p.requestCount} permintaan` : 'belum dipakai'}
                    </div>
                  </div>

                  {akanDihapus === p.id ? (
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-ink-600">Hapus {p.name}?</span>
                      <Button size="sm" variant="danger" loading={menghapus} onClick={() => hapus(p)}>
                        Hapus
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => setAkanDihapus(null)}>
                        Batal
                      </Button>
                    </div>
                  ) : (
                    <div className="flex items-center gap-1">
                      <Button size="sm" variant="ghost" onClick={() => mulaiUbah(p)} aria-label={`Ubah ${p.name}`}>
                        <Pencil className="h-3.5 w-3.5" />
                        Ubah
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={dipakai}
                        title={dipakai ? 'Periode yang sudah punya permintaan konfirmasi tidak bisa dihapus.' : undefined}
                        onClick={() => setAkanDihapus(p.id)}
                        aria-label={`Hapus ${p.name}`}
                        className="text-red-600 hover:bg-red-50 hover:text-red-700"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        Hapus
                      </Button>
                    </div>
                  )}
                </li>
              )
            })}
          </ul>
        )}
        <p className="mt-2 text-xs text-ink-400">
          Periode yang sudah punya permintaan konfirmasi tidak bisa dihapus, karena jawaban MC di sana menjadi
          dasar invoice renewal. Namanya dan tanggalnya tetap bisa diubah.
        </p>
      </div>
    </Modal>
  )
}
