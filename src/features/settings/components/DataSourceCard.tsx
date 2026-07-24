import { Database, HardDrive, Check } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui'
import {
  DATA_SOURCE_LABEL,
  getDataSource,
  setDataSource,
  type DataSource,
} from '@/services/dataSource'

interface Option {
  value: DataSource
  icon: React.ReactNode
  title: string
  hint: string
  detail: string
}

const OPTIONS: Option[] = [
  {
    value: 'mock',
    icon: <HardDrive className="h-5 w-5" />,
    title: DATA_SOURCE_LABEL.mock,
    hint: 'Berjalan sepenuhnya di browser',
    detail: 'Cocok untuk demo dan eksplorasi — tidak perlu backend maupun database.',
  },
  {
    value: 'api',
    icon: <Database className="h-5 w-5" />,
    title: DATA_SOURCE_LABEL.api,
    hint: 'Terhubung ke REST API Go',
    detail: 'Data nyata dari Postgres. Backend harus berjalan lebih dulu.',
  },
]

/**
 * Pemilih sumber data.
 *
 * Mengganti pilihan memuat ulang halaman: repository dirakit satu kali saat
 * modul dimuat, jadi menukarnya di tempat akan menyisakan sebagian layar
 * memakai sumber lama.
 */
export function DataSourceCard() {
  const active = getDataSource()

  return (
    <Card className="lg:col-span-2">
      <CardHeader
        title={
          <span className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-sky-50 text-sky-500">
              <Database className="h-5 w-5" />
            </span>
            Sumber Data
          </span>
        }
        subtitle="Pilih dari mana aplikasi membaca dan menulis data."
      />
      <CardBody className="space-y-4">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {OPTIONS.map((option) => {
            const isActive = option.value === active
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={isActive}
                onClick={() => !isActive && setDataSource(option.value)}
                className={`group relative rounded-xl border p-4 text-left transition ${
                  isActive
                    ? 'border-brand-500 bg-brand-50/50 ring-1 ring-brand-500'
                    : 'border-ink-200 hover:border-ink-300 hover:bg-ink-50'
                }`}
              >
                <span className="flex items-start gap-3">
                  <span
                    className={`mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg ${
                      isActive ? 'bg-brand-500 text-white' : 'bg-ink-100 text-ink-500'
                    }`}
                  >
                    {option.icon}
                  </span>
                  <span className="min-w-0">
                    <span className="flex items-center gap-1.5 font-semibold text-ink-900">
                      {option.title}
                      {isActive && <Check className="h-4 w-4 text-brand-500" />}
                    </span>
                    <span className="mt-0.5 block text-xs text-ink-500">{option.hint}</span>
                    <span className="mt-2 block text-xs leading-relaxed text-ink-400">
                      {option.detail}
                    </span>
                  </span>
                </span>
              </button>
            )
          })}
        </div>

        <p className="border-t border-ink-100 pt-3 text-xs text-ink-400">
          Mengganti sumber data akan <strong>memuat ulang halaman</strong> dan mengakhiri sesi
          login, karena akun di kedua mode berbeda.
        </p>
      </CardBody>
    </Card>
  )
}
