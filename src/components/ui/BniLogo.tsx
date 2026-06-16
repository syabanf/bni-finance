/**
 * Logo wordmark BNI (merah brand). Set `white` untuk dipakai di atas background gelap/merah.
 * Untuk mengganti dengan file PNG resmi: taruh di /public/bni-logo.png lalu ganti komponen ini
 * dengan <img src="/bni-logo.png" />.
 */
export function BniLogo({ className, white = false }: { className?: string; white?: boolean }) {
  const fill = white ? '#ffffff' : '#E2231A'
  return (
    <svg viewBox="0 0 112 40" className={className} role="img" aria-label="BNI">
      <text
        x="0"
        y="33"
        fontFamily="'Arial Black','Helvetica Neue',Arial,sans-serif"
        fontWeight="900"
        fontSize="40"
        letterSpacing="-3"
        fill={fill}
      >
        BNI
      </text>
      <text x="100" y="13" fontFamily="Arial, sans-serif" fontSize="10" fill={fill}>®</text>
    </svg>
  )
}
