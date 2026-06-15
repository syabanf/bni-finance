import { useEffect, useState } from 'react'
import { Save, UserPlus, RefreshCw, Info } from 'lucide-react'
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
import { formatCurrency, formatDateTime } from '@/lib/format'

export function SettingsPage() {
  const { toast } = useToast()
  const { data: fees, loading, reload } = useAsync<FeeSettings>(() => settingsService.getFees())

  const [registrationFee, setRegistrationFee] = useState(0)
  const [renewalFee, setRenewalFee] = useState(0)
  const [notes, setNotes] = useState('')
  const [saving, setSaving] = useState(false)

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
