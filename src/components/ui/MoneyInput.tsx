import { forwardRef, type InputHTMLAttributes } from 'react'
import { Input } from './Field'

interface MoneyInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  value: number
  onChange: (value: number) => void
}

/**
 * Input rupiah dengan pemisah ribuan: 18000000 tampil sebagai 18.000.000.
 *
 * `type="text"` dengan `inputMode="numeric"`, bukan `type="number"`. Input
 * angka bawaan peramban menolak titik pemisah dan menampilkan 18000000 apa
 * adanya, dan pada nominal delapan digit orang menghitung nol dengan jari.
 * Salah satu nol pada harga pendaftaran berarti tagihan sepuluh kali lipat
 * atau sepersepuluh, dan keduanya tercetak di invoice member.
 *
 * Yang disimpan tetap angka. Setiap ketikan dibersihkan dari apa pun selain
 * digit, lalu diformat ulang, jadi menempel "Rp 1.500.000" dari tempat lain
 * pun menghasilkan 1500000.
 */
export const MoneyInput = forwardRef<HTMLInputElement, MoneyInputProps>(
  ({ value, onChange, ...props }, ref) => (
    <Input
      ref={ref}
      type="text"
      inputMode="numeric"
      autoComplete="off"
      value={value ? value.toLocaleString('id-ID') : ''}
      onChange={(e) => onChange(Number(e.target.value.replace(/\D/g, '')) || 0)}
      {...props}
    />
  ),
)
MoneyInput.displayName = 'MoneyInput'
