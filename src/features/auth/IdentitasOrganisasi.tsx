import { ShieldCheck } from 'lucide-react'

/**
 * Siapa pemilik situs ini, ditulis di setiap halaman yang bisa dibuka tanpa
 * login.
 *
 * "BNI" juga singkatan Bank Negara Indonesia, salah satu merek yang paling
 * sering dipalsukan. Formulir kata sandi di domain yang memuat "bni" tanpa
 * penjelasan pemiliknya ditandai Google Safe Browsing sebagai phishing, dan
 * Chrome menutup seluruh situs dengan halaman peringatan merah.
 *
 * KONTAK kosong berarti barisnya tidak ditampilkan. Isi dengan kontak resmi
 * organisasi; jangan diisi alamat contoh, karena kontak yang tidak bisa
 * dihubungi justru menguatkan dugaan situs palsu.
 */
export const ORGANISASI = 'Business Network International Indonesia'

const KONTAK: { email: string; situs: string } = { email: '', situs: '' }

export function IdentitasOrganisasi() {
  return (
    <div className="mt-4 rounded-xl border border-ink-100 bg-white/70 px-4 py-3 text-center text-xs leading-relaxed text-ink-500">
      <p className="flex items-center justify-center gap-1.5 font-medium text-ink-700">
        <ShieldCheck className="h-3.5 w-3.5 shrink-0" />
        {ORGANISASI}
      </p>
      <p className="mt-1">
        Sistem internal untuk pengurus chapter. Tidak terkait Bank BNI (PT Bank Negara Indonesia)
        dan tidak pernah meminta PIN, OTP bank, atau nomor kartu.
      </p>
      {(KONTAK.email || KONTAK.situs) && (
        <p className="mt-1">
          Kontak resmi:{' '}
          {KONTAK.email && (
            <a className="font-medium text-brand-600 hover:underline" href={`mailto:${KONTAK.email}`}>
              {KONTAK.email}
            </a>
          )}
          {KONTAK.email && KONTAK.situs && ' · '}
          {KONTAK.situs && (
            <a
              className="font-medium text-brand-600 hover:underline"
              href={`https://${KONTAK.situs}`}
              target="_blank"
              rel="noreferrer"
            >
              {KONTAK.situs}
            </a>
          )}
        </p>
      )}
    </div>
  )
}
