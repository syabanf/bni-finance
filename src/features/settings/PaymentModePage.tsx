import { useEffect, useState } from 'react'
import { Info, Wallet, CreditCard, Mail, MessageCircle } from 'lucide-react'
import {
  Card,
  CardBody,
  LoadingState,
  PageHeader,
  useToast,
} from '@/components/ui'
import { getAppSetting, setAppSetting } from '@/services/appSettings'
import { isMockMode } from '@/services/dataSource'

const useMock = isMockMode()

/** Kanal pengiriman Paper.id — kunci app_settings yang dibaca server saat Send. */
const CHANNELS = [
  {
    key: 'paperid_send_email',
    label: 'Email',
    icon: Mail,
    hint: 'Paper.id mengirim invoice ke alamat email member. Member tanpa email dilewati.',
  },
  {
    key: 'paperid_send_whatsapp',
    label: 'WhatsApp',
    icon: MessageCircle,
    hint: 'Paper.id mengirim tautan pembayaran ke nomor WhatsApp member.',
  },
] as const

export function PaymentModePage() {
  const { toast } = useToast()
  const [selfPayment, setSelfPayment] = useState(false)
  const [channels, setChannels] = useState<Record<string, boolean>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    Promise.all([
      getAppSetting('self_payment_mode'),
      ...CHANNELS.map((c) => getAppSetting(c.key)),
    ])
      .then(([mode, ...values]) => {
        setSelfPayment(mode === 'true')
        setChannels(Object.fromEntries(CHANNELS.map((c, i) => [c.key, values[i] === 'true'])))
      })
      .finally(() => setLoading(false))
  }, [])

  const toggleChannel = async (key: string, label: string) => {
    const next = !channels[key]
    setChannels((c) => ({ ...c, [key]: next }))
    try {
      await setAppSetting(key, String(next))
      toast(next ? `${label} diaktifkan — member akan menerima invoice.` : `${label} dimatikan.`)
    } catch {
      setChannels((c) => ({ ...c, [key]: !next }))
      toast(`Gagal menyimpan pengaturan ${label}.`, 'error')
    }
  }

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

        {/* Pengiriman invoice — hanya berlaku pada jalur Paper.id. */}
        <Card className={selfPayment ? 'opacity-60' : undefined}>
          <CardBody className="p-6">
            <div className="mb-1 text-base font-semibold text-ink-900">Pengiriman Invoice</div>
            <p className="mb-4 text-sm text-ink-500">
              {selfPayment
                ? 'Tidak berlaku saat Self Payment Mode aktif — Paper.id tidak dipanggil, tautan pembayaran dibagikan manual dari halaman invoice.'
                : 'Kanal yang dipakai Paper.id untuk mengantar invoice ke member saat tombol Terbitkan ditekan.'}
            </p>

            <div className="space-y-3">
              {CHANNELS.map((c) => {
                const Icon = c.icon
                const on = channels[c.key] ?? false
                return (
                  <div key={c.key} className="flex items-start justify-between gap-4">
                    <div className="flex min-w-0 gap-3">
                      <span
                        className={`mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg ${
                          on ? 'bg-emerald-50 text-emerald-600' : 'bg-ink-100 text-ink-400'
                        }`}
                      >
                        <Icon className="h-4 w-4" />
                      </span>
                      <div className="min-w-0 leading-snug">
                        <div className="text-sm font-medium text-ink-900">{c.label}</div>
                        <div className="mt-0.5 text-xs text-ink-400">{c.hint}</div>
                      </div>
                    </div>
                    <button
                      role="switch"
                      aria-checked={on}
                      aria-label={`Kirim invoice via ${c.label}`}
                      disabled={selfPayment}
                      onClick={() => toggleChannel(c.key, c.label)}
                      className={`relative mt-1 inline-flex h-7 w-12 flex-shrink-0 items-center rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                        on ? 'bg-emerald-500' : 'bg-ink-200'
                      }`}
                    >
                      <span
                        className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
                          on ? 'translate-x-6' : 'translate-x-1'
                        }`}
                      />
                    </button>
                  </div>
                )
              })}
            </div>

            {!selfPayment && !channels.paperid_send_email && !channels.paperid_send_whatsapp && (
              <div className="mt-4 flex items-start gap-2 rounded-xl bg-amber-50 p-3 text-xs text-amber-800">
                <Info className="mt-0.5 h-4 w-4 flex-shrink-0" />
                Semua kanal mati — invoice tetap dibuat di Paper.id, tetapi member tidak
                menerima pemberitahuan apa pun. Aman untuk uji coba, bukan untuk operasional.
              </div>
            )}
            {useMock && (
              <div className="mt-3 text-xs text-ink-400">
                Pada Data Contoh tidak ada pesan yang benar-benar terkirim — pengaturannya hanya disimpan.
              </div>
            )}
          </CardBody>
        </Card>
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
