import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ChevronLeft, ChevronRight, Volume2, VolumeX, X } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { TourStep } from './tourSteps'
import { useNarration } from './useNarration'

/** Kotak sorotan pada layar, atau null bila langkahnya tidak menunjuk elemen. */
interface Spot {
  top: number
  left: number
  width: number
  height: number
}

const PAD = 8

function measure(anchor?: string): Spot | null {
  if (!anchor) return null
  const el = document.querySelector<HTMLElement>(`[data-tour="${anchor}"]`)
  if (!el) return null
  const r = el.getBoundingClientRect()
  if (r.width === 0 && r.height === 0) return null
  return { top: r.top - PAD, left: r.left - PAD, width: r.width + PAD * 2, height: r.height + PAD * 2 }
}

export function TourOverlay({
  steps,
  index,
  soundOn,
  onSound,
  onIndex,
  onClose,
}: {
  steps: TourStep[]
  index: number
  soundOn: boolean
  onSound: (on: boolean) => void
  onIndex: (i: number) => void
  onClose: () => void
}) {
  const step = steps[index]
  const [spot, setSpot] = useState<Spot | null>(null)
  // Tinggi panel diukur, bukan ditebak: isi narasinya berbeda-beda panjang,
  // dan angka tetap akan menempatkan panel menembus tepi layar pada langkah
  // yang teksnya paling panjang — tepat langkah yang paling perlu terbaca.
  const panelRef = useRef<HTMLDivElement>(null)
  const [panelH, setPanelH] = useState(240)
  const narration = useNarration(soundOn)

  // Ukur SETELAH elemen digulir ke layar, bukan sebelumnya: mengukur lebih dulu
  // menghasilkan koordinat posisi lama, dan sorotannya mendarat di tempat yang
  // salah persis pada langkah yang butuh digulir.
  useLayoutEffect(() => {
    const el = step?.anchor
      ? document.querySelector<HTMLElement>(`[data-tour="${step.anchor}"]`)
      : null
    el?.scrollIntoView({ block: 'center', behavior: 'smooth' })

    const sync = () => setSpot(measure(step?.anchor))
    sync()
    // Gulirannya halus, jadi posisi akhirnya baru stabil beberapa frame kemudian.
    const t = window.setTimeout(sync, 420)
    window.addEventListener('resize', sync)
    window.addEventListener('scroll', sync, true)
    return () => {
      window.clearTimeout(t)
      window.removeEventListener('resize', sync)
      window.removeEventListener('scroll', sync, true)
    }
  }, [step])

  // Ukur tinggi panel setiap kali isinya berganti.
  useLayoutEffect(() => {
    const h = panelRef.current?.offsetHeight
    if (h && h !== panelH) setPanelH(h)
  }, [step, panelH])

  // Narasikan tiap langkah. Judul dan isi dibaca sebagai satu kalimat supaya
  // tidak ada jeda janggal di antaranya.
  useEffect(() => {
    if (step) narration.speak(`${step.title}. ${step.body}`)
    // narration sengaja tidak masuk dependensi: identitasnya berubah tiap
    // render, dan memasukkannya akan mengulang narasi tanpa henti.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step, soundOn])

  // Escape menutup, panah berpindah — tur yang hanya bisa ditutup dengan mouse
  // menjebak pengguna keyboard.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      else if (e.key === 'ArrowRight') onIndex(Math.min(index + 1, steps.length - 1))
      else if (e.key === 'ArrowLeft') onIndex(Math.max(index - 1, 0))
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [index, steps.length, onIndex, onClose])

  if (!step) return null

  const last = index === steps.length - 1

  // Panel diletakkan di bawah sorotan bila muat, kalau tidak di atasnya, dan
  // kalau dua-duanya tidak muat ia dijepit agar tetap di dalam layar.
  const GAP = 12
  const MARGIN = 16
  const left = spot
    ? Math.max(MARGIN, Math.min(spot.left, window.innerWidth - 400 - MARGIN))
    : 0
  const below = spot ? spot.top + spot.height + GAP : 0
  const above = spot ? spot.top - panelH - GAP : 0
  const panelStyle: React.CSSProperties = spot
    ? below + panelH + MARGIN <= window.innerHeight
      ? { top: below, left }
      : above >= MARGIN
        ? { top: above, left }
        : { top: Math.max(MARGIN, window.innerHeight - panelH - MARGIN), left }
    : { top: '50%', left: '50%', transform: 'translate(-50%, -50%)' }

  return createPortal(
    <div className="fixed inset-0 z-[60]" role="dialog" aria-modal="true" aria-label="Panduan penggunaan">
      {/* Latar gelap. Bila ada sorotan, lubangnya dibuat dengan box-shadow raksasa
          — cara yang tetap tajam di layar mana pun tanpa SVG mask. */}
      {spot ? (
        <div
          className="pointer-events-none absolute rounded-xl ring-1 ring-white/70 transition-all duration-300 ease-out"
          style={{
            top: spot.top,
            left: spot.left,
            width: spot.width,
            height: spot.height,
            // Lebih terang daripada gelap pekat: tujuannya mengarahkan mata,
            // bukan memadamkan sisa halaman. Pengguna tetap perlu melihat
            // konteks di sekitar elemen yang sedang dijelaskan.
            boxShadow: '0 0 0 9999px rgba(15, 23, 42, 0.45)',
          }}
        />
      ) : (
        <div className="absolute inset-0 bg-ink-900/45" />
      )}

      {/* Klik di luar panel menutup tur. */}
      <button
        className="absolute inset-0 h-full w-full cursor-default"
        aria-label="Tutup panduan"
        onClick={onClose}
      />

      <div
        ref={panelRef}
        className="absolute w-[min(400px,calc(100vw-32px))] rounded-2xl bg-white p-5 shadow-2xl ring-1 ring-ink-900/5"
        style={panelStyle}
      >
        <div className="mb-2 flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <p className="mb-1 text-[11px] font-medium uppercase tracking-wide text-ink-400">
              Langkah {index + 1} dari {steps.length}
            </p>
            <h3 className="text-[15px] font-bold leading-snug text-ink-900">{step.title}</h3>
          </div>
          <button
            onClick={() => onSound(!soundOn)}
            className="rounded-lg p-1 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-600"
            aria-label={soundOn ? 'Matikan narasi suara' : 'Nyalakan narasi suara'}
            title={soundOn ? 'Matikan narasi suara' : 'Nyalakan narasi suara'}
          >
            {soundOn ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
          </button>
          <button
            onClick={onClose}
            className="rounded-lg p-1 text-ink-400 transition-colors hover:bg-ink-100 hover:text-ink-600"
            aria-label="Tutup panduan"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <p className="text-[13.5px] leading-[1.7] text-ink-600">{step.body}</p>

        {/* Titik kemajuan: memberi rasa "tinggal sedikit lagi" tanpa menambah
            angka lain yang harus dibaca. */}
        <div className="mt-4 flex items-center gap-1.5">
          {steps.map((_, i) => (
            <span
              key={i}
              className={cn(
                'h-1 rounded-full transition-all duration-300',
                i === index ? 'w-5 bg-brand-500' : i < index ? 'w-1.5 bg-brand-200' : 'w-1.5 bg-ink-200',
              )}
            />
          ))}
        </div>

        {soundOn && narration.supported && !narration.hasIndonesian && (
          <p className="mt-2 text-[11px] leading-snug text-amber-700">
            Peramban ini belum punya suara Bahasa Indonesia, jadi narasinya memakai suara bawaan.
          </p>
        )}
        {soundOn && !narration.supported && (
          <p className="mt-2 text-[11px] leading-snug text-amber-700">
            Peramban ini tidak mendukung narasi suara. Panduannya tetap bisa dibaca.
          </p>
        )}

        <div className="mt-3 flex items-center gap-2">
          <button
            onClick={() => onIndex(index - 1)}
            disabled={index === 0}
            className={cn(
              'inline-flex items-center gap-1 rounded-xl px-2.5 py-1.5 text-xs font-medium transition-colors',
              index === 0 ? 'text-ink-300' : 'text-ink-600 hover:bg-ink-100',
            )}
          >
            <ChevronLeft className="h-3.5 w-3.5" />
            Kembali
          </button>
          <button
            onClick={() => (last ? onClose() : onIndex(index + 1))}
            className="ml-auto inline-flex items-center gap-1 rounded-xl bg-brand-500 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-brand-600"
          >
            {last ? 'Selesai' : 'Lanjut'}
            {!last && <ChevronRight className="h-3.5 w-3.5" />}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  )
}
