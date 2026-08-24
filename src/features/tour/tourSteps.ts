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
          'Halaman ini adalah ringkasan keuangan keanggotaan BNI. ' +
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
  {
    path: '/urgent',
    label: 'Perlu Tindakan',
    steps: [
      {
        title: 'Daftar kerja hari ini',
        body:
          'Halaman ini mengumpulkan dua hal yang tidak sebaiknya menunggu: ' +
          'tagihan yang sudah lewat jatuh tempo, dan keanggotaan yang sebentar lagi habis. ' +
          'Kalau Anda hanya sempat membuka satu halaman pagi ini, bukalah yang ini.',
      },
      {
        anchor: 'urgent-overdue',
        title: 'Tagihan yang terlambat',
        body:
          'Setiap baris menunjukkan sudah berapa lama tagihan itu lewat jatuh tempo. ' +
          'Tombol Ingatkan mengirim pengingat lewat Paper.id, jadi member menerimanya ' +
          'dari saluran yang sama dengan tagihan aslinya, dan pengingat itu tercatat. ' +
          'Anda tidak perlu mengirim pesan sendiri dari nomor pribadi.',
      },
      {
        anchor: 'urgent-renewal',
        title: 'Keanggotaan yang akan habis',
        body:
          'Bagian ini menampilkan member yang masa keanggotaannya mendekati akhir. ' +
          'Menerbitkan tagihan perpanjangan lebih awal memberi mereka waktu membayar ' +
          'sebelum keanggotaannya benar-benar lewat, dan itu jauh lebih mudah ' +
          'daripada menagih setelah terlambat.',
      },
    ],
  },
  {
    path: '/notifications',
    label: 'Notifikasi',
    steps: [
      {
        title: 'Pemberitahuan sistem',
        body:
          'Halaman ini mengumpulkan hal-hal yang sistem anggap perlu Anda ketahui, ' +
          'misalnya tagihan yang mendekati jatuh tempo atau keanggotaan yang perlu diperpanjang. ' +
          'Isinya dihitung dari data yang sama dengan halaman lain, jadi tidak akan ada ' +
          'pemberitahuan tentang sesuatu yang sudah Anda selesaikan.',
      },
      {
        anchor: 'notifications-list',
        title: 'Menindaklanjuti',
        body:
          'Menekan sebuah pemberitahuan membawa Anda ke tempat yang bisa ditindaklanjuti, ' +
          'bukan sekadar menandainya sudah dibaca. ' +
          'Yang belum dibaca ditandai lebih tegas supaya mudah dibedakan sekilas.',
      },
    ],
  },
  {
    path: '/profile',
    label: 'Profil',
    steps: [
      {
        title: 'Akun Anda',
        body: 'Halaman ini untuk memeriksa data akun sendiri dan mengganti kata sandi.',
      },
      {
        anchor: 'profile-account',
        title: 'Informasi akun',
        body:
          'Nama dan email akun Anda beserta perannya. ' +
          'Peran menentukan apa yang bisa Anda lakukan: admin bisa menerbitkan dan mengirim tagihan, ' +
          'sementara peran pengguna hanya bisa melihat dan mengekspor.',
      },
      {
        anchor: 'profile-password',
        title: 'Mengganti kata sandi',
        body:
          'Isi kata sandi lama dan yang baru, lalu simpan. ' +
          'Setelah tersimpan, kata sandi lama langsung tidak berlaku, ' +
          'jadi pastikan yang baru sudah Anda catat di tempat aman.',
      },
    ],
  },
  {
    path: '/invoices/new',
    label: 'Buat Invoice',
    steps: [
      {
        title: 'Menerbitkan tagihan',
        body:
          'Formulir ini disusun berurutan dari atas ke bawah. ' +
          'Isi dari langkah pertama, dan bagian berikutnya menyesuaikan pilihan Anda.',
      },
      {
        anchor: 'invoice-new-form',
        title: 'Mengisi tagihan',
        body:
          'Pilih jenis tagihan lebih dulu, apakah pendaftaran baru atau perpanjangan, ' +
          'karena itu menentukan nominal dan periode yang disarankan. ' +
          'Lalu pilih membernya, periksa nominalnya, dan tentukan jatuh tempo. ' +
          'Tanggal jatuh tempo inilah yang akan muncul di dokumen yang diterima member.',
      },
      {
        title: 'Setelah disimpan',
        body:
          'Tagihan tersimpan sebagai draft dan belum sampai ke member. ' +
          'Anda bisa mengirimnya langsung dari sini, atau menyimpannya dulu ' +
          'lalu mengirim beberapa sekaligus dari daftar invoice. ' +
          'Draft masih bisa diubah; yang sudah terkirim tidak.',
      },
    ],
  },
  {
    path: '/invoices/renewal-due',
    label: 'Renewal Due',
    steps: [
      {
        title: 'Menyiapkan perpanjangan',
        body:
          'Halaman ini memperlihatkan member yang keanggotaannya akan berakhir, ' +
          'supaya tagihan perpanjangan bisa disiapkan sebelum masanya lewat.',
      },
      {
        anchor: 'renewal-range',
        title: 'Mengatur rentang waktu',
        body:
          'Ubah rentang tanggal untuk melihat lebih jauh ke depan atau lebih dekat. ' +
          'Menagih terlalu awal membuat member bingung, menagih terlalu lambat ' +
          'membuat keanggotaan sempat kosong. Rentang inilah yang mengatur keseimbangannya.',
      },
      {
        anchor: 'renewal-list',
        title: 'Menerbitkan sekaligus',
        body:
          'Pilih beberapa member, lalu terbitkan tagihan perpanjangan untuk semuanya. ' +
          'Semua tagihan itu dibuat sebagai draft dulu, jadi Anda masih sempat memeriksanya ' +
          'sebelum satu pun dikirim.',
      },
    ],
  },
  {
    path: '/invoices/:id',
    label: 'Detail Invoice',
    steps: [
      {
        title: 'Satu tagihan, seluruh riwayatnya',
        body:
          'Halaman ini menampilkan segalanya tentang satu tagihan: isinya, statusnya, ' +
          'tautan pembayarannya, dan setiap perubahan yang pernah terjadi padanya.',
      },
      {
        anchor: 'detail-status',
        title: 'Status dan tindakan',
        body:
          'Tindakan yang tersedia mengikuti status. Tagihan draft bisa diubah dan dikirim. ' +
          'Yang sudah terkirim bisa diingatkan atau dicatat pembayarannya. ' +
          'Yang sudah lunas atau dibatalkan adalah catatan tertutup — nominal dan periodenya ' +
          'tidak bisa ditulis ulang setelah fakta.',
      },
      {
        anchor: 'detail-audit',
        title: 'Jejak audit',
        body:
          'Setiap perubahan status tercatat di sini beserta waktu dan pelakunya, ' +
          'termasuk setiap pengingat yang pernah dikirim. ' +
          'Kalau seorang member bertanya sudah berapa kali ditagih, jawabannya ada di bagian ini.',
      },
    ],
  },
  {
    path: '/members',
    label: 'Member',
    steps: [
      {
        title: 'Data keanggotaan',
        body:
          'Semua member beserta chapter, status, dan kelengkapan kontaknya. ' +
          'Data ini yang dipakai saat tagihan dikirim, jadi ketepatannya berpengaruh langsung ' +
          'pada apakah pesannya sampai.',
      },
      {
        anchor: 'member-filters',
        title: 'Menemukan yang bermasalah',
        body:
          'Saringan kelengkapan kontak adalah yang paling berguna di sini. ' +
          'Paper.id mewajibkan nomor telepon, jadi member tanpa nomor akan ditolak ' +
          'saat tagihannya diterbitkan. Menyaring mereka sekarang jauh lebih baik ' +
          'daripada menemukannya saat batch penagihan sedang berjalan.',
      },
      {
        anchor: 'member-table',
        title: 'Membuka satu member',
        body:
          'Menekan sebuah baris membuka halaman detailnya, ' +
          'lengkap dengan riwayat tagihan member tersebut.',
      },
      {
        title: 'Mengubah kontak',
        body:
          'Satu hal yang perlu diketahui: Paper.id menyimpan data kontak member di sisi mereka. ' +
          'Kalau Anda mengubah nomor telepon atau email, tagihan berikutnya akan mengirim ' +
          'kontak yang baru, dan sistem sudah menangani perbedaan itu secara otomatis.',
      },
    ],
  },
  {
    path: '/members/:id',
    label: 'Detail Member',
    steps: [
      {
        title: 'Profil satu member',
        body:
          'Halaman ini menggabungkan data member dengan seluruh riwayat tagihannya, ' +
          'sehingga Anda bisa menjawab pertanyaan mereka tanpa berpindah halaman.',
      },
      {
        anchor: 'member-detail-info',
        title: 'Data dan kontak',
        body:
          'Nama, chapter, status keanggotaan, dan kontak. ' +
          'Kontak inilah yang dipakai Paper.id untuk mengantar tagihan, ' +
          'jadi kalau member mengeluh tidak menerima apa-apa, mulailah memeriksa dari sini.',
      },
      {
        anchor: 'member-detail-invoices',
        title: 'Riwayat tagihan',
        body:
          'Seluruh tagihan member ini beserta statusnya. ' +
          'Berguna saat mereka bertanya sudah membayar apa saja, atau saat Anda perlu ' +
          'memastikan tidak ada tagihan ganda untuk periode yang sama.',
      },
    ],
  },
  {
    path: '/chapters',
    label: 'Chapter',
    steps: [
      {
        title: 'Pengelompokan member',
        body:
          'Chapter adalah kelompok wilayah tempat member bernaung. ' +
          'Setiap member harus punya chapter, dan laporan bisa disaring per chapter.',
      },
      {
        anchor: 'chapter-list',
        title: 'Menambah dan mengubah',
        body:
          'Tambahkan chapter baru, atau ubah namanya di sini. ' +
          'Chapter yang masih punya member tidak bisa dihapus — ' +
          'kalau bisa, member-membernya akan kehilangan induk tanpa pemberitahuan.',
      },
    ],
  },
  {
    path: '/payments',
    label: 'Pembayaran',
    steps: [
      {
        title: 'Catatan uang masuk',
        body:
          'Setiap pembayaran yang tercatat, baik yang datang otomatis dari Paper.id ' +
          'maupun yang Anda catat manual setelah menerima transfer.',
      },
      {
        anchor: 'payment-table',
        title: 'Membaca satu pembayaran',
        body:
          'Tiap baris menunjukkan tagihan mana yang dilunasi, berapa nominalnya, ' +
          'kapan, dan lewat cara apa. ' +
          'Mencatat pembayaran akan melunasi tagihannya dalam satu langkah, ' +
          'jadi tidak ada status yang perlu Anda ubah terpisah.',
      },
    ],
  },
  {
    path: '/reports',
    label: 'Laporan',
    steps: [
      {
        title: 'Laporan keuangan',
        body:
          'Halaman ini merangkum penerimaan dan tunggakan dalam rentang waktu yang Anda pilih, ' +
          'siap untuk dibawa ke rapat atau diarsipkan.',
      },
      {
        anchor: 'report-range',
        title: 'Menentukan periode',
        body:
          'Semua angka dan grafik di bawah mengikuti rentang tanggal ini. ' +
          'Ubah rentangnya lebih dulu sebelum membaca angkanya, ' +
          'karena angka yang benar untuk periode yang salah tetap menyesatkan.',
      },
      {
        anchor: 'report-export',
        title: 'Mengekspor',
        body:
          'Ekspor tersedia dalam Excel, CSV, dan PDF. ' +
          'Yang diekspor selalu mengikuti rentang dan saringan yang sedang tampil, ' +
          'bukan seluruh data — jadi periksa dulu apa yang di layar sebelum menekan ekspor.',
      },
    ],
  },
  {
    path: '/settings',
    label: 'Pengaturan',
    steps: [
      {
        title: 'Pengaturan operasional',
        body:
          'Nilai-nilai di halaman ini memengaruhi tagihan yang diterbitkan setelahnya. ' +
          'Tagihan yang sudah ada tidak ikut berubah, dan itu memang disengaja: ' +
          'mengubah nominal tagihan yang sudah dikirim akan membuat catatan tidak lagi ' +
          'cocok dengan dokumen yang diterima member.',
      },
      {
        anchor: 'settings-fee',
        title: 'Nominal biaya',
        body:
          'Biaya pendaftaran dan perpanjangan. Nilai inilah yang otomatis terisi ' +
          'saat Anda membuat tagihan baru, dan masih bisa diubah per tagihan bila perlu.',
      },
      {
        anchor: 'settings-schedule',
        title: 'Jadwal penagihan',
        body:
          'Dua angka ini mengatur ritme penagihan. Yang pertama menentukan berapa hari ' +
          'sebelum keanggotaan berakhir draft perpanjangan mulai disiapkan. ' +
          'Yang kedua menentukan berapa hari setelah diterbitkan sebuah tagihan jatuh tempo. ' +
          'Menagih terlalu awal membingungkan member, terlalu lambat membuat keanggotaan ' +
          'sempat kosong — dua angka inilah yang mengatur keseimbangannya.',
      },
    ],
  },
  {
    path: '/settings/sync',
    label: 'Sinkronisasi',
    steps: [
      {
        title: 'Menarik data member',
        body:
          'Halaman ini menarik data keanggotaan dari sistem BNI VM ' +
          'supaya daftar member di sini tidak perlu dijaga manual.',
      },
      {
        anchor: 'sync-source',
        title: 'Dari mana datanya',
        body:
          'Bagian ini menunjukkan sumber datanya beserta status sambungannya. ' +
          'Kalau sinkronisasi gagal, periksa di sini lebih dulu sebelum mencoba lagi.',
      },
      {
        anchor: 'sync-run',
        title: 'Member dan chapter, terpisah',
        body:
          'Keduanya disinkronkan sendiri-sendiri, dan setiap kartu menunjukkan ' +
          'kapan terakhir kali datanya ditarik. ' +
          'Menjalankan dua kali berturut-turut tidak menggandakan data — ' +
          'yang sudah ada diperbarui, bukan ditambahkan lagi, jadi aman mengulang ' +
          'bila Anda ragu apakah yang tadi berhasil. ' +
          'Member yang hilang dari sumber dinonaktifkan, bukan dihapus, ' +
          'karena tagihan lama mereka harus tetap bisa ditelusuri.',
      },
    ],
  },
  {
    path: '/api-console',
    label: 'Konsol API',
    steps: [
      {
        title: 'Alat teknis',
        body:
          'Konsol ini untuk menguji endpoint API secara langsung. ' +
          'Ditujukan bagi yang sedang menelusuri masalah teknis, ' +
          'bukan untuk pekerjaan penagihan sehari-hari.',
      },
      {
        anchor: 'console-endpoint',
        title: 'Memilih endpoint',
        body:
          'Pilih endpoint, dan parameternya terisi otomatis dengan data yang benar-benar ada, ' +
          'jadi Anda tidak perlu mencari id yang valid lebih dulu. ' +
          'Isian body ditampilkan sebagai kolom berlabel, bukan JSON mentah.',
      },
      {
        anchor: 'console-send',
        title: 'Hati-hati dengan yang mengubah data',
        body:
          'Permintaan dari konsol ini berjalan sungguhan. ' +
          'Menjalankan pengiriman tagihan dari sini benar-benar mengirim ke member ' +
          'dan membakar satu nomor tagihan di Paper.id secara permanen. ' +
          'Untuk membaca data, konsol ini aman sepenuhnya.',
      },
    ],
  },
]

/**
 * Cocokkan rute, termasuk yang berparameter.
 *
 * Pencocokan persis saja tidak cukup: `/invoices/:id` tidak akan pernah sama
 * dengan `/invoices/abc-123`, sehingga halaman detail — yang justru paling
 * banyak tombolnya — tidak akan pernah punya panduan.
 *
 * Rute yang lebih spesifik diperiksa lebih dulu, karena `/invoices/new` dan
 * `/invoices/:id` sama-sama cocok dengan dua segmen: tanpa urutan ini, halaman
 * Buat Invoice akan mendapat panduan halaman Detail.
 */
export function tourFor(pathname: string): Tour | null {
  const segs = pathname.split('/').filter(Boolean)
  const cocok = TOURS.filter((t) => {
    const pat = t.path.split('/').filter(Boolean)
    if (pat.length !== segs.length) return false
    return pat.every((p, i) => p.startsWith(':') || p === segs[i])
  })
  if (cocok.length === 0) return null
  // Yang paling sedikit parameternya = paling spesifik.
  return cocok.sort(
    (a, b) =>
      a.path.split('/').filter((x) => x.startsWith(':')).length -
      b.path.split('/').filter((x) => x.startsWith(':')).length,
  )[0]
}
