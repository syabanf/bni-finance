/**
 * Langkah tur per halaman.
 *
 * Setiap langkah menunjuk elemen lewat atribut `data-tour`, BUKAN selector CSS.
 * Selector yang menyusuri kelas Tailwind atau struktur DOM akan patah pada
 * perubahan tata letak yang paling sepele, dan patahnya diam-diam: tur tetap
 * berjalan, hanya menyorot ruang kosong. Penanda `data-tour` membuat
 * hubungannya eksplisit, dan `scripts/check-tour-anchors.mjs` menjaga agar
 * tidak ada langkah yang menunjuk penanda yang sudah tidak ada.
 *
 * CARA MENULIS ISINYA
 *
 * `body` dibacakan narator apa adanya, jadi tulislah seperti orang menjelaskan
 * kepada rekan yang baru duduk di sebelahnya — kalimat utuh, mengalir, dan
 * menjawab "kenapa", bukan hanya "ini tombol apa". Daftar fitur yang dibacakan
 * keras terdengar seperti mesin; penjelasan yang menyebutkan alasannya
 * terdengar seperti orang.
 *
 * Hindari singkatan, tanda kurung, dan simbol — semuanya terdengar aneh saat
 * disintesis menjadi suara.
 *
 * Empat sampai enam langkah per halaman. Lebih dari itu, orang berhenti
 * menyimak sebelum sampai akhir.
 */

export interface TourStep {
  /** Nilai atribut data-tour pada elemen yang disorot. Kosong = di tengah layar. */
  anchor?: string
  title: string
  /** Dibacakan narator apa adanya, jadi tulis sebagai kalimat utuh yang mengalir. */
  body: string
}

export interface Tour {
  /** Rute tempat tur ini berlaku. */
  path: string
  label: string
  steps: TourStep[]
}

export const TOURS: Tour[] = [
  {
    path: '/dashboard',
    label: 'Dashboard',
    steps: [
      {
        title: 'Selamat datang',
        body:
          'Halaman ini adalah ringkasan keuangan keanggotaan BNI Grow. ' +
          'Semua yang Anda lihat di sini dihitung langsung dari data tagihan yang sebenarnya, ' +
          'jadi tidak ada angka yang disimpan terpisah lalu berbeda dengan kenyataan. ' +
          'Mari saya tunjukkan bagian-bagiannya. Tekan Lanjut untuk berpindah, ' +
          'atau tutup panduan ini kapan saja bila Anda sudah paham.',
      },
      {
        anchor: 'dashboard-summary',
        title: 'Empat angka yang perlu Anda pantau',
        body:
          'Total memberitahu berapa banyak tagihan yang sudah diterbitkan beserta nilainya. ' +
          'Lunas adalah yang sudah dibayar. Outstanding adalah yang sudah dikirim tetapi belum dibayar, ' +
          'dan inilah uang yang masih di luar. Terlambat adalah bagian dari outstanding yang sudah melewati ' +
          'jatuh tempo, dan itu yang biasanya paling perlu ditindaklanjuti hari ini. ' +
          'Kalau angka terlambat naik, mulailah dari halaman Mendesak.',
      },
      {
        anchor: 'topbar-search',
        title: 'Mencari satu tagihan',
        body:
          'Kalau seorang member menghubungi Anda dan menyebut nomor tagihannya, ' +
          'ketik nomor itu di sini lalu tekan Enter. Anda juga bisa mengetik namanya. ' +
          'Anda akan dibawa langsung ke daftar tagihan yang sudah tersaring, ' +
          'jadi tidak perlu menggulir mencari satu per satu.',
      },
      {
        anchor: 'topbar-notifications',
        title: 'Yang butuh perhatian',
        body:
          'Lonceng ini menandai hal yang sebaiknya tidak menunggu, ' +
          'misalnya tagihan yang mendekati jatuh tempo atau keanggotaan yang perlu diperpanjang. ' +
          'Angka merah di sudutnya menunjukkan berapa yang belum Anda baca.',
      },
      {
        title: 'Itu saja untuk halaman ini',
        body:
          'Tombol tanda tanya di bagian atas tersedia di setiap halaman yang punya panduan, ' +
          'jadi tekan lagi kapan pun Anda ragu. ' +
          'Kalau Anda lebih suka membaca tanpa suara, matikan lewat ikon pengeras suara di panduan ini.',
      },
    ],
  },
  {
    path: '/invoices',
    label: 'Daftar Invoice',
    steps: [
      {
        title: 'Tempat pekerjaan penagihan berlangsung',
        body:
          'Di halaman inilah tagihan diterbitkan, dikirim ke member, dan diingatkan bila belum dibayar. ' +
          'Seluruh pengiriman berjalan lewat Paper.id, dan tidak ada satu pun jalur manual, ' +
          'sehingga setiap pesan yang sampai ke member selalu meninggalkan catatan.',
      },
      {
        anchor: 'invoice-filters',
        title: 'Menyaring lebih dulu',
        body:
          'Kartu-kartu ini bukan sekadar hiasan. Menekannya menyaring tabel di bawah menurut status, ' +
          'jadi kalau Anda ingin melihat semua yang belum dibayar, tekan Outstanding. ' +
          'Satu hal yang perlu diingat: ekspor selalu mengikuti hasil saringan yang sedang tampil, ' +
          'bukan seluruh data. Jadi saringlah dulu, baru ekspor.',
      },
      {
        anchor: 'invoice-new',
        title: 'Menerbitkan tagihan baru',
        body:
          'Tagihan yang baru dibuat berstatus draft. Draft belum sampai ke mana-mana, ' +
          'dan member belum tahu apa pun tentangnya. Ia baru benar-benar terkirim setelah didorong ke Paper.id. ' +
          'Jadi Anda aman menyiapkan banyak draft dulu, memeriksanya, baru mengirim semuanya sekaligus.',
      },
      {
        anchor: 'invoice-table',
        title: 'Mengirim banyak sekaligus',
        body:
          'Centang beberapa baris, lalu kirim sekaligus. Pengirimannya berjalan satu per satu, ' +
          'dan tiap tagihan bisa memakan waktu setengah menit karena Paper.id perlu menyiapkan dokumennya. ' +
          'Kalau ada satu yang gagal, misalnya karena member belum punya nomor telepon, ' +
          'sisanya tetap dikirim. Anda akan diberi tahu berapa yang berhasil, berapa yang gagal, ' +
          'dan tagihan mana saja beserta alasannya.',
      },
      {
        title: 'Mengingatkan yang belum bayar',
        body:
          'Tombol lonceng pada tagihan yang sudah terkirim mengirim pengingat, juga lewat Paper.id. ' +
          'Satu hal yang perlu Anda ketahui: Paper.id tidak menyediakan cara mengirim ulang dokumen yang sama, ' +
          'jadi setiap pengingat menerbitkan dokumen baru dengan nomor turunan. ' +
          'Artinya member akan melihat beberapa dokumen bernilai sama di riwayat mereka. ' +
          'Itu wajar, dan nomor tagihan di sistem kita sendiri tidak berubah.',
      },
      {
        title: 'Kalau ada yang perlu ditelusuri',
        body:
          'Setiap pengiriman dan setiap pengingat tercatat di halaman Blackbox, ' +
          'lengkap dengan apa yang dikirim dan apa jawaban Paper.id. ' +
          'Jadi kalau suatu saat seorang member bilang tidak menerima apa-apa, ' +
          'jawabannya ada di sana, bukan di ingatan siapa pun.',
      },
    ],
  },
  {
    path: '/blackbox',
    label: 'Blackbox',
    steps: [
      {
        title: 'Kotak hitam integrasi',
        body:
          'Halaman ini merekam setiap percakapan antara sistem kita dan Paper.id. ' +
          'Namanya kotak hitam karena fungsinya sama dengan yang di pesawat: ' +
          'Anda tidak membukanya saat semuanya lancar, tetapi ketika ada yang salah, ' +
          'di sinilah jawabannya. Riwayatnya tersimpan di database, ' +
          'jadi tetap ada meski server dinyalakan ulang.',
      },
      {
        anchor: 'blackbox-filters',
        title: 'Mempersempit pencarian',
        body:
          'Saring menurut integrasi atau arah panggilan. ' +
          'Kalau Anda sedang menelusuri masalah, cara tercepat adalah menampilkan yang gagal saja, ' +
          'karena biasanya hanya beberapa baris dan langsung menunjuk penyebabnya.',
      },
      {
        anchor: 'blackbox-list',
        title: 'Membaca satu rekaman',
        body:
          'Setiap baris memuat empat hal: apa yang kita kirim, apa yang dijawab Paper.id, ' +
          'kode statusnya, dan berapa lama prosesnya. ' +
          'Kode dua ratus atau dua ratus satu berarti berhasil. ' +
          'Empat ratus sekian berarti ada yang salah pada permintaan kita, ' +
          'misalnya nomor tagihan yang sudah terpakai. ' +
          'Lima ratus sekian berarti masalahnya di sisi Paper.id.',
      },
      {
        title: 'Dua baris untuk satu pengiriman',
        body:
          'Anda akan melihat dua rekaman untuk setiap pengiriman, dan itu memang disengaja. ' +
          'Satu mencatat apa yang dilakukan sistem kita, satu lagi mencatat percakapan sebenarnya dengan Paper.id. ' +
          'Keduanya berguna: yang pertama menjawab apakah permintaannya sah, ' +
          'yang kedua menjawab apa kata Paper.id.',
      },
    ],
  },
]

export function tourFor(pathname: string): Tour | null {
  return TOURS.find((t) => t.path === pathname) ?? null
}
