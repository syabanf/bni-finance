import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { HelpCircle } from 'lucide-react'
import { TourOverlay } from './TourOverlay'
import { tourFor } from './tourSteps'

const SOUND_KEY = 'bni.tour.sound'

/**
 * Pemicu panduan. Hanya muncul di halaman yang benar-benar punya tur —
 * tombol bantuan yang membuka panduan kosong lebih buruk daripada tidak ada.
 */
export function TourButton() {
  const { pathname } = useLocation()
  const tour = tourFor(pathname)
  const [open, setOpen] = useState(false)
  const [index, setIndex] = useState(0)
  const [soundOn, setSoundOn] = useState(() => localStorage.getItem(SOUND_KEY) !== 'off')

  // Berpindah halaman saat tur terbuka akan menyorot elemen yang sudah tidak
  // ada, jadi turnya ditutup mengikuti navigasi.
  useEffect(() => {
    setOpen(false)
    setIndex(0)
  }, [pathname])

  const setSound = (on: boolean) => {
    setSoundOn(on)
    localStorage.setItem(SOUND_KEY, on ? 'on' : 'off')
  }

  if (!tour) return null

  return (
    <>
      <button
        onClick={() => {
          setIndex(0)
          setOpen(true)
        }}
        data-tour="help-button"
        className="rounded-xl p-2 text-ink-500 transition-colors hover:bg-ink-100 hover:text-ink-700"
        aria-label={`Panduan penggunaan halaman ${tour.label}`}
        title={`Panduan: ${tour.label}`}
      >
        <HelpCircle className="h-5 w-5" />
      </button>

      {open && (
        <TourOverlay
          steps={tour.steps}
          index={index}
          soundOn={soundOn}
          onSound={setSound}
          onIndex={setIndex}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  )
}
