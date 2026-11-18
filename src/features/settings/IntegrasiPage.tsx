import { PageHeader } from '@/components/ui'
import { BniVmCard } from './components/BniVmCard'
import { DataSourceCard } from './components/DataSourceCard'
import { SUMBER_TERKUNCI } from '@/services/dataSource'
import { JUDUL_INTEGRASI, TARIK_VM_TAMPIL } from '@/lib/fitur'

/**
 * Alamat API, token, dan pemilihan sumber data.
 *
 * Dipindah keluar dari halaman Pengaturan Biaya. Halaman itu dibuka orang yang
 * mengurus harga, pengingat, dan teks invoice, dan di tengahnya dulu berdiri
 * kolom "Alamat API" beserta sebuah token. Dua hal terjadi karenanya: yang
 * mengurus harga membaca layar yang separuh isinya bukan urusannya, dan yang
 * memegang token melihatnya di tempat yang tidak dijaga kebiasaan apa pun.
 *
 * Di sini ia bertetangga dengan Konsol API dan Blackbox, yaitu tempat yang
 * memang dibuka saat orang sedang mengurus sambungan ke sistem lain.
 */
export function IntegrasiPage() {
  return (
    <div>
      <PageHeader
        title={JUDUL_INTEGRASI}
        description={
          TARIK_VM_TAMPIL ? 'Sambungan ke sistem luar.' : 'Dari mana aplikasi membaca datanya.'
        }
      />
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        {TARIK_VM_TAMPIL && <BniVmCard />}
        {!SUMBER_TERKUNCI && <DataSourceCard />}
      </div>
    </div>
  )
}
