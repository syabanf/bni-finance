import { Search } from 'lucide-react'
import { DateRangeFilter, Input, Select } from '@/components/ui'

export interface RentangTanggal {
  label: string
  dari: string
  sampai: string
  onDari: (v: string) => void
  onSampai: (v: string) => void
}

/** Pencarian nomor invoice atau nama member, saring chapter, dan rentang tanggal. */
export function Saringan({
  cari,
  onCari,
  chapterId,
  onChapter,
  chapters,
  tanggal,
}: {
  cari: string
  onCari: (v: string) => void
  chapterId: string
  onChapter: (v: string) => void
  chapters: { id: string; displayName: string }[]
  tanggal: RentangTanggal
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-ink-100 px-4 py-3">
      <div className="relative min-w-[14rem] flex-1">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
        <Input
          value={cari}
          onChange={(e) => onCari(e.target.value)}
          placeholder="Cari nomor invoice atau nama member"
          aria-label="Cari"
          className="pl-9"
        />
      </div>
      {/* ST hanya menerima chapternya sendiri dari server, jadi pilihannya
          tidak pernah memuat chapter lain. */}
      <Select value={chapterId} onChange={(e) => onChapter(e.target.value)} aria-label="Chapter" className="w-52">
        <option value="all">Semua Chapter</option>
        {chapters.map((c) => (
          <option key={c.id} value={c.id}>
            {c.displayName}
          </option>
        ))}
      </Select>
      <DateRangeFilter
        label={tanggal.label}
        from={tanggal.dari}
        to={tanggal.sampai}
        onFrom={tanggal.onDari}
        onTo={tanggal.onSampai}
        onReset={() => {
          tanggal.onDari('')
          tanggal.onSampai('')
        }}
      />
    </div>
  )
}
