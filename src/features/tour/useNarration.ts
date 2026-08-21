import { useCallback, useEffect, useRef, useState } from 'react'

/**
 * Narasi panduan lewat Web Speech API — bawaan peramban, tanpa berkas audio
 * dan tanpa layanan luar.
 *
 * DUA HAL YANG MUDAH SALAH, DAN KEDUANYA DIJAGA DI SINI.
 *
 * Pertama, `speechSynthesis` hidup di level WINDOW, bukan komponen. Ia terus
 * berbicara setelah komponennya di-unmount, setelah pengguna berpindah
 * halaman, bahkan setelah tur ditutup — suara yang menjelaskan halaman yang
 * sudah tidak ada di layar. Karena itu setiap jalur keluar memanggil cancel().
 *
 * Kedua, daftar suara dimuat ASINKRON di sebagian peramban: memanggil
 * getVoices() terlalu cepat mengembalikan array kosong, dan narasinya jatuh ke
 * suara bawaan berbahasa Inggris yang membaca teks Indonesia. Karena itu
 * pemilihan suara ditunda sampai `voiceschanged` menyala.
 */

const LANG = 'id-ID'

function pickVoice(): SpeechSynthesisVoice | null {
  const voices = window.speechSynthesis?.getVoices?.() ?? []
  if (voices.length === 0) return null
  return (
    voices.find((v) => v.lang === LANG) ??
    voices.find((v) => v.lang?.toLowerCase().startsWith('id')) ??
    null
  )
}

export interface Narration {
  /** Peramban ini mendukung sintesis suara sama sekali. */
  supported: boolean
  /** Ada suara berbahasa Indonesia. Bila false, narasi tetap jalan memakai suara bawaan. */
  hasIndonesian: boolean
  speaking: boolean
  speak: (text: string) => void
  stop: () => void
}

export function useNarration(enabled: boolean): Narration {
  const supported = typeof window !== 'undefined' && 'speechSynthesis' in window
  const [voice, setVoice] = useState<SpeechSynthesisVoice | null>(null)
  const [speaking, setSpeaking] = useState(false)
  // Menyimpan ucapan yang sedang berjalan supaya handler-nya bisa dilepas —
  // tanpa ini, onend milik ucapan lama tetap menyalakan state komponen baru.
  const current = useRef<SpeechSynthesisUtterance | null>(null)

  useEffect(() => {
    if (!supported) return
    const load = () => setVoice(pickVoice())
    load()
    window.speechSynthesis.addEventListener('voiceschanged', load)
    return () => window.speechSynthesis.removeEventListener('voiceschanged', load)
  }, [supported])

  const stop = useCallback(() => {
    if (!supported) return
    if (current.current) {
      current.current.onend = null
      current.current.onerror = null
      current.current = null
    }
    window.speechSynthesis.cancel()
    setSpeaking(false)
  }, [supported])

  const speak = useCallback(
    (text: string) => {
      if (!supported || !enabled || !text.trim()) return
      // Batalkan dulu: memanggil speak() dua kali menumpuk antrean, sehingga
      // menekan "Lanjut" cepat-cepat membuat beberapa langkah dibacakan
      // bersamaan.
      stop()

      const u = new SpeechSynthesisUtterance(text)
      u.lang = LANG
      if (voice) u.voice = voice
      u.rate = 0.98
      u.onend = () => setSpeaking(false)
      u.onerror = () => setSpeaking(false)
      current.current = u
      setSpeaking(true)
      window.speechSynthesis.speak(u)
    },
    [supported, enabled, voice, stop],
  )

  // Jaring pengaman terakhir: apa pun yang membuat hook ini hilang — unmount,
  // navigasi, hot reload — harus menghentikan suaranya.
  useEffect(() => stop, [stop])

  return {
    supported,
    hasIndonesian: voice !== null,
    speaking,
    speak,
    stop,
  }
}
