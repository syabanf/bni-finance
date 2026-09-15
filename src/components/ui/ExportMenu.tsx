import { useEffect, useRef, useState } from 'react'
import { ChevronDown, Download, FileSpreadsheet, FileText } from 'lucide-react'
import { cn } from '@/lib/cn'

interface ExportMenuProps {
  onExcel?: () => void
  onPdf?: () => void
  disabled?: boolean
  label?: string
}

/** Lebar menu, dipakai juga untuk mengukur sisi mana yang muat. */
const LEBAR_MENU = 192 // w-48

/** "Export ▾" button with Excel / CSV / PDF options (each optional). */
export function ExportMenu({ onExcel, onPdf, disabled, label = 'Export' }: ExportMenuProps) {
  const [open, setOpen] = useState(false)
  /**
   * Menu dibuka ke kanan saat membuka ke kiri akan keluar layar.
   *
   * Sebelumnya ia selalu `right-0`, yang berarti tepi kanan menu menempel ke
   * tepi kanan tombol dan badannya memanjang ke KIRI. Di layar ponsel, tombol
   * Export berdiri dekat tepi kiri, jadi menu selebar 192px itu mulai di luar
   * layar: diukur pada layar 375px, tepi kirinya jatuh di -54px dan separuh
   * tulisannya terpotong jadi "xport Excel".
   *
   * Sisi yang dipakai diukur, bukan ditebak lewat media query. Tombol yang
   * sama dipakai di header yang rata kanan maupun rata kiri, dan aturan
   * berdasar lebar layar akan membenarkan salah satunya sambil merusak yang
   * lain.
   */
  const [bukaKeKanan, setBukaKeKanan] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const pick = (fn: () => void) => {
    setOpen(false)
    fn()
  }

  const toggle = () => {
    if (!open && ref.current) {
      const kotak = ref.current.getBoundingClientRect()
      // 8px sisa supaya menunya tidak menempel persis di tepi layar.
      setBukaKeKanan(kotak.right - LEBAR_MENU < 8)
    }
    setOpen((o) => !o)
  }

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        disabled={disabled}
        onClick={toggle}
        title={disabled ? 'Tidak ada data untuk diekspor pada filter saat ini.' : 'Pilih format export'}
        className="inline-flex items-center gap-2 rounded-xl border border-ink-200 bg-white px-3.5 py-2 text-sm font-medium text-ink-700 transition-colors hover:bg-ink-50 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <Download className="h-4 w-4" />
        {label}
        <ChevronDown className={cn('h-4 w-4 text-ink-400 transition-transform', open && 'rotate-180')} />
      </button>
      {open && (
        <div
          className={cn(
            'absolute top-full z-20 mt-2 w-48 overflow-hidden rounded-xl border border-ink-100 bg-white py-1.5 shadow-card-hover animate-fade-in',
            bukaKeKanan ? 'left-0' : 'right-0',
          )}
        >
          {onExcel && (
            <button
              type="button"
              onClick={() => pick(onExcel)}
              className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-ink-600 hover:bg-ink-50"
            >
              <FileSpreadsheet className="h-4 w-4 text-emerald-600" />
              Export Excel
            </button>
          )}
          {onPdf && (
            <button
              type="button"
              onClick={() => pick(onPdf)}
              className="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm text-ink-600 hover:bg-ink-50"
            >
              <FileText className="h-4 w-4 text-red-600" />
              Export PDF
            </button>
          )}
        </div>
      )}
    </div>
  )
}
