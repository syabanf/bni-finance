import { ChevronDown } from 'lucide-react'
import { Badge, Card, CardBody, CardHeader, PageHeader } from '@/components/ui'
import { ALUR, FAQ, type AlurPanduan } from './panduanData'

export function PanduanPage() {
  return (
    <div>
      <PageHeader
        title="Panduan"
        description="Video dan langkah untuk alur utama, lalu jawaban atas pertanyaan yang sering muncul."
      />

      <div className="space-y-4">
        {ALUR.map((alur) => (
          <KartuAlur key={alur.id} alur={alur} />
        ))}
      </div>

      <h2 id="faq" className="mb-3 mt-8 scroll-mt-24 text-lg font-bold text-ink-900">
        Pertanyaan yang sering muncul
      </h2>
      <div className="space-y-4">
        {FAQ.map((grup) => (
          <Card key={grup.kelompok}>
            <CardHeader title={grup.kelompok} />
            <div className="divide-y divide-ink-100 border-t border-ink-100">
              {grup.butir.map((b) => (
                <details key={b.tanya} className="group px-5 py-3.5">
                  <summary className="flex cursor-pointer list-none items-start justify-between gap-3 text-sm font-medium text-ink-800 [&::-webkit-details-marker]:hidden">
                    {b.tanya}
                    <ChevronDown className="mt-0.5 h-4 w-4 shrink-0 text-ink-400 transition-transform group-open:rotate-180" />
                  </summary>
                  <p className="mt-2 text-sm leading-relaxed text-ink-600">{b.jawab}</p>
                </details>
              ))}
            </div>
          </Card>
        ))}
      </div>
    </div>
  )
}

function KartuAlur({ alur }: { alur: AlurPanduan }) {
  return (
    <Card className="overflow-hidden">
      <div className="grid lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <div className="bg-ink-900">
          {/* preload none: lima video tidak boleh terunduh hanya karena halaman dibuka. */}
          <video
            controls
            playsInline
            preload="none"
            poster={`/video-panduan/${alur.id}.jpg`}
            aria-label={`Video: ${alur.judul}`}
            className="aspect-video w-full"
          >
            <source src={`/video-panduan/${alur.id}.webm`} type="video/webm" />
          </video>
        </div>
        <CardBody>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-base font-bold text-ink-900">{alur.judul}</h2>
            {alur.khususAdmin && (
              <Badge tone="gray" dot={false}>
                Khusus Admin
              </Badge>
            )}
          </div>
          <p className="mt-1 text-sm text-ink-500">{alur.ringkas}</p>
          <ol className="mt-3 list-decimal space-y-1.5 pl-5 text-sm leading-relaxed text-ink-700">
            {alur.langkah.map((l) => (
              <li key={l}>{l}</li>
            ))}
          </ol>
        </CardBody>
      </div>
    </Card>
  )
}
