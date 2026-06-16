import { useEffect, useState } from 'react'
import { Info, Wallet, CreditCard } from 'lucide-react'
import {
  Card,
  CardBody,
  LoadingState,
  PageHeader,
  useToast,
} from '@/components/ui'
import { getAppSetting, setAppSetting } from '@/services/supabase/settingsRepository'

const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

export function PaymentModePage() {
  const { toast } = useToast()
  const [selfPayment, setSelfPayment] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (useMock) {
      setLoading(false)
      return
    }
    getAppSetting('self_payment_mode')
      .then((v) => setSelfPayment(v === 'true'))
      .finally(() => setLoading(false))
  }, [])

  const toggle = async () => {
    const next = !selfPayment
    setSelfPayment(next)
    setSaving(true)
    try {
      await setAppSetting('self_payment_mode', String(next))
      toast(next ? 'Payment Gateway Xendit diaktifkan.' : 'Beralih ke integrasi Paper.id.')
    } catch {
      setSelfPayment(!next)
      toast('Gagal menyimpan konfigurasi.', 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingState label="Memuat konfigurasi…" />

  return (
    <div>
      <PageHeader
        title="Metode Pembayaran"
        description="Pilih cara member membayar tagihan: Payment Gateway Xendit atau Paper.id."
      />

      <div className="mx-auto max-w-2xl space-y-5">
        <Card>
          <CardBody className="flex items-center justify-between gap-4 p-6">
            <div className="leading-snug">
              <div className="text-base font-semibold text-ink-900">Self Payment Mode (Xendit)</div>
              <div className="mt-0.5 text-sm text-ink-400">
                {selfPayment
                  ? 'AKTIF — member bayar mandiri via Virtual Account & QRIS.'
                  : 'NONAKTIF — pembayaran memakai integrasi Paper.id.'}
              </div>
            </div>
            <button
              role="switch"
              aria-checked={selfPayment}
              disabled={saving || useMock}
              onClick={toggle}
              className={`relative inline-flex h-8 w-14 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
                selfPayment ? 'bg-emerald-500' : 'bg-ink-200'
              }`}
            >
              <span
                className={`inline-block h-6 w-6 transform rounded-full bg-white shadow transition-transform ${
                  selfPayment ? 'translate-x-7' : 'translate-x-1'
                }`}
              />
            </button>
          </CardBody>
        </Card>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <ModeCard
            active={selfPayment}
            icon={<Wallet className="h-5 w-5" />}
            title="ON — Xendit"
            tone="emerald"
            points={[
              'Member bayar sendiri (self payment)',
              'Virtual Account: BCA, BNI, Mandiri, BRI',
              'QRIS (nominal ≤ Rp 10 juta)',
              'Otomatis Lunas via webhook',
            ]}
          />
          <ModeCard
            active={!selfPayment}
            icon={<CreditCard className="h-5 w-5" />}
            title="OFF — Paper.id"
            tone="blue"
            points={[
              'Integrasi Paper.id (seperti semula)',
              'Link pembayaran Paper.id dikirim ke member',
              'Cocok bila belum pakai gateway sendiri',
            ]}
          />
        </div>

        {!useMock && selfPayment && (
          <div className="flex items-start gap-2 rounded-xl bg-emerald-50 p-3 text-xs text-emerald-700">
            <Info className="mt-0.5 h-4 w-4 flex-shrink-0" />
            Pastikan Secret Key & Callback Token Xendit sudah dikonfigurasi di server (Supabase secrets).
          </div>
        )}
      </div>
    </div>
  )
}

function ModeCard({
  active,
  icon,
  title,
  points,
  tone,
}: {
  active: boolean
  icon: React.ReactNode
  title: string
  points: string[]
  tone: 'emerald' | 'blue'
}) {
  const ring = active
    ? tone === 'emerald'
      ? 'border-emerald-300 ring-2 ring-emerald-100'
      : 'border-blue-300 ring-2 ring-blue-100'
    : 'border-ink-200'
  const badge = tone === 'emerald' ? 'bg-emerald-50 text-emerald-500' : 'bg-blue-50 text-blue-500'
  return (
    <div className={`rounded-2xl border bg-white p-5 transition ${ring}`}>
      <div className="mb-3 flex items-center justify-between">
        <span className={`flex h-9 w-9 items-center justify-center rounded-lg ${badge}`}>{icon}</span>
        {active && <span className="text-xs font-semibold uppercase tracking-wide text-ink-400">Aktif</span>}
      </div>
      <div className="mb-2 font-semibold text-ink-900">{title}</div>
      <ul className="space-y-1.5">
        {points.map((p) => (
          <li key={p} className="flex gap-2 text-sm text-ink-600">
            <span className="mt-1.5 h-1 w-1 flex-shrink-0 rounded-full bg-ink-300" />
            {p}
          </li>
        ))}
      </ul>
    </div>
  )
}
