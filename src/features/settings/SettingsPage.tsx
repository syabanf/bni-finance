import { useEffect, useState } from 'react'
import { Save, UserPlus, RefreshCw, Info, Clock, Wallet } from 'lucide-react'
import type { FeeSettings } from '@/types'
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Field,
  Input,
  LoadingState,
  PageHeader,
  Textarea,
  useToast,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { settingsService } from '@/services'
import { getAppSetting, setAppSetting } from '@/services/supabase/settingsRepository'
import { formatCurrency, formatDateTime } from '@/lib/format'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

export function SettingsPage() {
  const { toast } = useToast()
  const { data: fees, loading, reload } = useAsync<FeeSettings>(() => settingsService.getFees())

  const [registrationFee, setRegistrationFee] = useState(0)
  const [renewalFee, setRenewalFee] = useState(0)
  const [notes, setNotes] = useState('')
  const [saving, setSaving] = useState(false)

  // Invoice timing
  const [draftDaysBefore, setDraftDaysBefore] = useState(30)
  const [dueDaysAfter, setDueDaysAfter] = useState(30)
  const [savingTiming, setSavingTiming] = useState(false)

  // Self Payment Mode (Xendit)
  const [selfPayment, setSelfPayment] = useState(false)
  const [savingSelfPayment, setSavingSelfPayment] = useState(false)

  useEffect(() => {
    if (useMock) return
    getAppSetting('invoice_draft_days_before').then(v => { if (v) setDraftDaysBefore(Number(v)) })
    getAppSetting('invoice_due_days_after').then(v => { if (v) setDueDaysAfter(Number(v)) })
    getAppSetting('self_payment_mode').then(v => setSelfPayment(v === 'true'))
  }, [])

  const toggleSelfPayment = async () => {
    const next = !selfPayment
    setSelfPayment(next)
    setSavingSelfPayment(true)
    try {
      await setAppSetting('self_payment_mode', String(next))
      toast(next ? 'Self Payment Mode diaktifkan (Xendit).' : 'Self Payment Mode dimatikan.')
    } catch {
      setSelfPayment(!next) // rollback
      toast('Gagal menyimpan Self Payment Mode.', 'error')
    } finally {
      setSavingSelfPayment(false)
    }
  }

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
      setRegistrationFee(fees.registrationFee)
      setRenewalFee(fees.renewalFee)
      setNotes(fees.notes ?? '')
    }
  }, [fees])

  const dirty =
    !!fees &&
    (registrationFee !== fees.registrationFee ||
      renewalFee !== fees.renewalFee ||
      notes !== (fees.notes ?? ''))

  const handleSave = async () => {
    setSaving(true)
    try {
      await settingsService.updateFees({ registrationFee, renewalFee, notes: notes.trim() || undefined })
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
          <Card>
            <CardHeader title="Nominal Biaya" subtitle="Nilai ini otomatis terisi saat membuat invoice baru." />
            <CardBody className="space-y-5">
              <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
                <FeeInput
                  icon={<UserPlus className="h-5 w-5" />}
                  label="Biaya Pendaftaran"
                  hint="Visitor → Member (berlaku 1 tahun)"
                  value={registrationFee}
                  onChange={setRegistrationFee}
                />
                <FeeInput
                  icon={<RefreshCw className="h-5 w-5" />}
                  label="Biaya Renewal"
                  hint="Perpanjangan tahunan member"
                  value={renewalFee}
                  onChange={setRenewalFee}
                />
              </div>

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

        {/* Invoice Timing */}
        {!useMock && (
          <Card className="lg:col-span-2">
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

        {/* Self Payment Mode */}
        {!useMock && (
          <Card className="lg:col-span-2">
            <CardHeader
              title={
                <span className="flex items-center gap-2.5">
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-50 text-emerald-500">
                    <Wallet className="h-5 w-5" />
                  </span>
                  Self Payment Mode
                </span>
              }
              subtitle="Aktifkan pembayaran mandiri via Xendit (Virtual Account & QRIS) langsung di aplikasi."
            />
            <CardBody className="space-y-4">
              <div className="flex items-center justify-between gap-4 rounded-xl border border-ink-200 p-4">
                <div className="leading-snug">
                  <div className="text-sm font-semibold text-ink-900">
                    {selfPayment ? 'Aktif — pembayaran via Xendit' : 'Nonaktif — penandaan lunas manual'}
                  </div>
                  <div className="text-xs text-ink-400">
                    {selfPayment
                      ? 'Member dapat membayar sendiri lewat VA (BCA, BNI, Mandiri, BRI) atau QRIS.'
                      : 'Invoice Outstanding ditandai lunas manual oleh admin.'}
                  </div>
                </div>
                <button
                  role="switch"
                  aria-checked={selfPayment}
                  disabled={savingSelfPayment}
                  onClick={toggleSelfPayment}
                  className={`relative inline-flex h-7 w-12 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
                    selfPayment ? 'bg-emerald-500' : 'bg-ink-200'
                  }`}
                >
                  <span
                    className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
                      selfPayment ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>
              <div className="flex items-start gap-2 rounded-xl bg-emerald-50 p-3 text-xs text-emerald-700">
                <Info className="mt-0.5 h-4 w-4 flex-shrink-0" />
                Pastikan Secret Key & Callback Token Xendit sudah dikonfigurasi di server (Supabase secrets) agar pembayaran berfungsi.
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
  value,
  onChange,
}: {
  icon: React.ReactNode
  label: string
  hint: string
  value: number
  onChange: (v: number) => void
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
      <div className="relative">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-ink-400">
          Rp
        </span>
        <Input
          type="number"
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          min={0}
          step={50000}
          className="pl-9 text-base font-semibold"
        />
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
