import { useEffect, useState } from 'react'
import { Save, UserPlus, RefreshCw, Info, Clock } from 'lucide-react'
import type { FeeSettings } from '@/types'
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Field,
  Input,
  MoneyInput,
  LoadingState,
  PageHeader,
  Textarea,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { settingsService } from '@/services'
import { rupiahDari } from '@/lib/kurs'
import { getAppSetting, setAppSetting } from '@/services/appSettings'
import { formatCurrency, formatDateTime } from '@/lib/format'
import { PaperProdukCard } from './components/PaperProdukCard'
import { ReminderCard } from './components/ReminderCard'


export function SettingsPage() {
  const { toast } = useToast()
  const { data: fees, loading, reload } = useAsync<FeeSettings>(() => settingsService.getFees())

  const [registrationFeeUsd, setRegistrationFeeUsd] = useState(0)
  const [renewalFeeUsd, setRenewalFeeUsd] = useState(0)
  const [usdRate, setUsdRate] = useState(0)
  const [notes, setNotes] = useState('')
  // Rupiah yang AKAN disimpan, dihitung dengan aturan yang sama seperti server.
  // Ditampilkan sebelum Simpan ditekan, supaya akibat dari angka yang baru
  // diketik terlihat lebih dulu.
  const registrationFee = rupiahDari(registrationFeeUsd, usdRate)
  const renewalFee = rupiahDari(renewalFeeUsd, usdRate)
  const [saving, setSaving] = useState(false)

  // Invoice timing
  const [draftDaysBefore, setDraftDaysBefore] = useState(30)
  const [dueDaysAfter, setDueDaysAfter] = useState(30)
  const [savingTiming, setSavingTiming] = useState(false)

  useEffect(() => {
    // Berlaku di kedua mode: mock membaca dari localStorage, API dari server.
    getAppSetting('invoice_draft_days_before').then(v => { if (v) setDraftDaysBefore(Number(v)) })
    getAppSetting('invoice_due_days_after').then(v => { if (v) setDueDaysAfter(Number(v)) })
  }, [])

  const saveTiming = async () => {
    setSavingTiming(true)
    try {
      await setAppSetting('invoice_draft_days_before', String(draftDaysBefore))
      await setAppSetting('invoice_due_days_after', String(dueDaysAfter))
      toast('Konfigurasi timing invoice berhasil disimpan.')
    } catch {
      toast('Gagal menyimpan konfigurasi timing.', 'error')
    } finally {
      setSavingTiming(false)
    }
  }

  useEffect(() => {
    if (fees) {
      setRegistrationFeeUsd(fees.registrationFeeUsd)
      setRenewalFeeUsd(fees.renewalFeeUsd)
      setUsdRate(fees.usdRate)
      setNotes(fees.notes ?? '')
    }
  }, [fees])

  const dirty =
    !!fees &&
    (registrationFeeUsd !== fees.registrationFeeUsd ||
      renewalFeeUsd !== fees.renewalFeeUsd ||
      usdRate !== fees.usdRate ||
      notes !== (fees.notes ?? ''))

  const handleSave = async () => {
    setSaving(true)
    try {
      await settingsService.updateFees({
        registrationFeeUsd,
        renewalFeeUsd,
        usdRate,
        notes: notes.trim() || undefined,
      })
      toast('Pengaturan biaya berhasil disimpan.')
      reload()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Gagal menyimpan.', 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading || !fees) return <LoadingState label="Memuat pengaturan…" />

  return (
    <div>
      <PageHeader
        title="Pengaturan Biaya"
        description="Konfigurasi nominal biaya pendaftaran dan renewal keanggotaan."
      />

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <Card data-tour="settings-fee">
            <CardHeader title="Nominal Biaya" subtitle="Nilai ini otomatis terisi saat membuat invoice baru." />
            <CardBody className="space-y-5">
              <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
                <FeeInput
                  icon={<UserPlus className="h-5 w-5" />}
                  label="Biaya Pendaftaran"
                  hint="Visitor → Member (berlaku 1 tahun)"
                  usd={registrationFeeUsd}
                  onChange={setRegistrationFeeUsd}
                  rupiah={registrationFee}
                />
                <FeeInput
                  icon={<RefreshCw className="h-5 w-5" />}
                  label="Biaya Renewal"
                  hint="Perpanjangan tahunan member"
                  usd={renewalFeeUsd}
                  onChange={setRenewalFeeUsd}
                  rupiah={renewalFee}
                />
              </div>

              {/* Satu kurs untuk kedua harga. Mengubahnya mengubah kedua
                  Rupiah sekaligus, dan itu memang maksudnya: BNI menetapkan
                  harga dalam Dollar, dan Rupiah hanya terjemahannya. */}
              <Field
                label="Kurs USD → IDR"
                hint="Rupiah per satu dolar, diisi manual. Kedua harga Rupiah dihitung ulang dari kurs ini."
              >
                <div className="relative sm:w-64">
                  <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-ink-400">
                    Rp
                  </span>
                  <MoneyInput value={usdRate} onChange={setUsdRate} className="pl-9 font-semibold" />
                </div>
              </Field>

              <Field label="Catatan" hint="Catatan internal mengenai kebijakan biaya.">
                <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} />
              </Field>

              <div className="flex items-center justify-between border-t border-ink-100 pt-4">
                <span className="text-xs text-ink-400">
                  Terakhir diubah {formatDateTime(fees.updatedAt)}
                </span>
                <Button onClick={handleSave} loading={saving} disabled={!dirty}>
                  <Save className="h-4 w-4" />
                  Simpan Perubahan
                </Button>
              </div>
            </CardBody>
          </Card>
        </div>

        <ReminderCard />

        <PaperProdukCard />

        {/* Invoice Timing — nilainya bertahan di localStorage pada mode mock,
            jadi tidak ada alasan menyembunyikannya dari demo. */}
        {(
          <Card data-tour="settings-schedule" className="lg:col-span-2">
            <CardHeader
              title={
                <span className="flex items-center gap-2.5">
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-50 text-violet-500">
                    <Clock className="h-5 w-5" />
                  </span>
                  Konfigurasi Timing Invoice
                </span>
              }
              subtitle="Atur kapan draft dibuat dan berapa lama jatuh tempo setelah invoice dikirim."
            />
            <CardBody className="space-y-5">
              <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
                <div className="rounded-xl border border-ink-200 p-4">
                  <div className="mb-1 text-sm font-semibold text-ink-900">Buat Draft Sebelum Renewal</div>
                  <div className="mb-3 text-xs text-ink-400">Invoice draft otomatis dibuat N hari sebelum renewal date member</div>
                  <div className="flex items-center gap-2">
                    <Input
                      type="number"
                      value={draftDaysBefore}
                      onChange={e => setDraftDaysBefore(Math.max(1, Number(e.target.value)))}
                      min={1}
                      max={90}
                      className="w-24 text-center font-semibold"
                    />
                    <span className="text-sm text-ink-500">hari sebelum renewal</span>
                  </div>
                </div>
                <div className="rounded-xl border border-ink-200 p-4">
                  <div className="mb-1 text-sm font-semibold text-ink-900">Jatuh Tempo Setelah Dikirim</div>
                  <div className="mb-3 text-xs text-ink-400">Due date invoice = tanggal kirim + N hari</div>
                  <div className="flex items-center gap-2">
                    <Input
                      type="number"
                      value={dueDaysAfter}
                      onChange={e => setDueDaysAfter(Math.max(1, Number(e.target.value)))}
                      min={1}
                      max={90}
                      className="w-24 text-center font-semibold"
                    />
                    <span className="text-sm text-ink-500">hari setelah dikirim</span>
                  </div>
                </div>
              </div>
              <div className="flex items-start gap-2 rounded-xl bg-violet-50 p-3 text-xs text-violet-700">
                <Info className="mt-0.5 h-4 w-4 flex-shrink-0" />
                Perubahan hanya berlaku untuk invoice yang dibuat setelah disimpan. Invoice yang sudah terbit tidak berubah.
              </div>
              <div className="flex justify-end border-t border-ink-100 pt-4">
                <Button onClick={saveTiming} loading={savingTiming}>
                  <Save className="h-4 w-4" />
                  Simpan Timing
                </Button>
              </div>
            </CardBody>
          </Card>
        )}

      {/* Preview */}
        <Card className="h-fit">
          <CardHeader title="Pratinjau" />
          <CardBody className="space-y-3">
            <PreviewRow label="Pendaftaran" value={formatCurrency(registrationFee)} tone="brand" />
            <PreviewRow label="Renewal" value={formatCurrency(renewalFee)} tone="violet" />
            <div className="flex items-start gap-2 rounded-xl bg-blue-50 p-3 text-xs text-blue-700">
              <Info className="mt-0.5 h-4 w-4 flex-shrink-0" />
              Perubahan biaya hanya berlaku untuk invoice yang dibuat setelah disimpan. Invoice lama tidak berubah.
            </div>
          </CardBody>
        </Card>
      </div>
    </div>
  )
}

function FeeInput({
  icon,
  label,
  hint,
  usd,
  onChange,
  rupiah,
}: {
  icon: React.ReactNode
  label: string
  hint: string
  usd: number
  onChange: (v: number) => void
  rupiah: number
}) {
  return (
    <div className="rounded-xl border border-ink-200 p-4">
      <div className="mb-3 flex items-center gap-2.5">
        <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-50 text-brand-500">
          {icon}
        </span>
        <div className="leading-tight">
          <div className="text-sm font-semibold text-ink-900">{label}</div>
          <div className="text-xs text-ink-400">{hint}</div>
        </div>
      </div>
      {/* Yang diketik Dollar; Rupiah di bawahnya hanya dibaca. BNI menetapkan
          harga dalam Dollar, dan Rupiah adalah hasil kurs, bukan angka yang
          berdiri sendiri. Dua kotak yang sama-sama bisa diketik adalah dua
          sumber untuk satu angka yang suatu saat tidak sama. */}
      <div className="relative">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-ink-400">
          $
        </span>
        <Input
          type="number"
          inputMode="decimal"
          min={0}
          step="0.01"
          value={usd}
          onChange={(e) => onChange(Math.max(0, Number(e.target.value) || 0))}
          className="pl-8 text-base font-semibold"
        />
      </div>
      <div className="mt-2 text-sm text-ink-600">
        = <span className="font-semibold text-ink-900">{formatCurrency(rupiah)}</span>
      </div>
    </div>
  )
}

function PreviewRow({ label, value, tone }: { label: string; value: string; tone: 'brand' | 'violet' }) {
  return (
    <div className="flex items-center justify-between rounded-xl bg-ink-50 px-4 py-3">
      <span className="flex items-center gap-2 text-sm text-ink-600">
        <span className={`h-2 w-2 rounded-full ${tone === 'brand' ? 'bg-brand-500' : 'bg-violet-500'}`} />
        {label}
      </span>
      <span className="font-bold text-ink-900">{value}</span>
    </div>
  )
}
