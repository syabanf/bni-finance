import type { InvoiceType, RincianBiaya } from '@/types'
import { Field, Input, MoneyInput } from '@/components/ui'
import { formatCurrency } from '@/lib/format'
import { barisBiaya, jumlahBaris, ppnUsd } from '@/lib/rincianBiaya'

const dolar = (n: number) => `$${n.toLocaleString('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`

/**
 * Penyunting rincian harga per komponen, di dalam kartu Nominal Biaya.
 *
 * Selama kotak centangnya mati, harga tetap satu angka Dollar per tipe dan
 * komponen di sini tidak berlaku. Mencentangnya lalu menekan Simpan membuat
 * kedua harga diturunkan dari komponen, dan invoice baru dikirim ke Paper.id
 * dengan satu baris per komponen.
 */
export function RincianBiayaSection({
  rincian,
  onChange,
  kurs,
}: {
  rincian: RincianBiaya
  onChange: (r: RincianBiaya) => void
  kurs: number
}) {
  const ubah = (bagian: Partial<RincianBiaya>) => onChange({ ...rincian, ...bagian })
  const angka = (v: string) => Math.max(0, Number(v) || 0)

  return (
    <div data-testid="rincian-biaya" className="space-y-4 rounded-xl border border-ink-200 p-4">
      <label className="flex cursor-pointer items-start gap-2.5">
        <input
          type="checkbox"
          checked={rincian.aktif}
          onChange={(e) => ubah({ aktif: e.target.checked })}
          className="mt-0.5 h-4 w-4 accent-bni-red"
        />
        <span>
          <span className="block text-sm font-semibold text-ink-900">Rinci harga per komponen</span>
          <span className="block text-xs leading-relaxed text-ink-500">
            Harga pendaftaran dan renewal dihitung dari komponen di bawah, dan invoice baru memuat
            satu baris per komponen.
          </span>
        </span>
      </label>

      {rincian.aktif && (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Registration Fee (USD)" hint="Hanya untuk pendaftaran.">
              <Input
                type="number"
                inputMode="decimal"
                min={0}
                step="0.01"
                value={rincian.pendaftaranUsd}
                onChange={(e) => ubah({ pendaftaranUsd: angka(e.target.value) })}
              />
            </Field>
            <Field label="Membership Fee (USD)" hint="Untuk pendaftaran dan renewal.">
              <Input
                type="number"
                inputMode="decimal"
                min={0}
                step="0.01"
                value={rincian.keanggotaanUsd}
                onChange={(e) => ubah({ keanggotaanUsd: angka(e.target.value) })}
              />
            </Field>
            <Field label="VAT (%)" hint="Dihitung dari komponen Dollar sebelum pajak.">
              <Input
                type="number"
                inputMode="decimal"
                min={0}
                max={100}
                step="0.01"
                value={rincian.ppnPersen}
                onChange={(e) => ubah({ ppnPersen: Math.min(100, angka(e.target.value)) })}
              />
            </Field>
            <Field label="Online Training (Rp)" hint="Hanya untuk pendaftaran. Tidak kena VAT.">
              <MoneyInput value={rincian.training} onChange={(v) => ubah({ training: v })} />
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <TabelRincian judul="Pendaftaran" tipe="registration" rincian={rincian} kurs={kurs} />
            <TabelRincian judul="Renewal" tipe="renewal" rincian={rincian} kurs={kurs} />
          </div>
        </>
      )}
    </div>
  )
}

/** Baris yang akan tercetak di invoice satu tipe, pada kurs yang sedang diisi. */
function TabelRincian({
  judul,
  tipe,
  rincian,
  kurs,
}: {
  judul: string
  tipe: InvoiceType
  rincian: RincianBiaya
  kurs: number
}) {
  const baris = barisBiaya(rincian, tipe, kurs)
  const usd: Record<string, number> = {
    registration: rincian.pendaftaranUsd,
    membership: rincian.keanggotaanUsd,
    vat: ppnUsd(rincian, tipe),
  }

  return (
    <div className="rounded-xl bg-ink-50 p-3">
      <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-400">{judul}</div>
      <table className="w-full text-sm">
        <tbody>
          {baris.map((b) => (
            <tr key={b.kode}>
              <td className="py-1 pr-2 text-ink-700">{b.nama}</td>
              <td className="whitespace-nowrap py-1 pr-2 text-right text-xs text-ink-400">
                {b.kode in usd ? dolar(usd[b.kode]) : ''}
              </td>
              <td className="whitespace-nowrap py-1 text-right text-ink-900">{formatCurrency(b.jumlah)}</td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr className="border-t border-ink-200 font-semibold">
            <td className="pt-2 text-ink-900" colSpan={2}>
              Total
            </td>
            <td className="whitespace-nowrap pt-2 text-right text-ink-900">
              {formatCurrency(jumlahBaris(baris))}
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  )
}
