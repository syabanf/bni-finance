/**
 * Isi halaman Panduan. Dipisah dari tampilannya supaya mengubah jawaban tidak
 * perlu menyentuh tata letak.
 *
 * Berkas video ada di public/video-panduan dan direkam oleh scripts/rekam-panduan.cjs.
 * `id` di sini menjadi nama berkasnya. Rekam ulang saat tampilan alurnya berubah.
 */

export interface AlurPanduan {
  id: string
  judul: string
  ringkas: string
  khususAdmin?: boolean
  langkah: string[]
}

export const ALUR: AlurPanduan[] = [
  {
    id: 'impor-member',
    judul: 'Impor member dari Business Network International Connect',
    ringkas: 'Dua laporan Business Network International Connect mengisi data member dan jatuh temponya.',
    khususAdmin: true,
    langkah: [
      'Buka Pengaturan, lalu Impor Data. Biarkan Chapter tujuan pada pilihan Dari laporan Business Network International Connect.',
      'Unggah Chapter Roster Report, lalu tekan Tinjau. Chapter dibaca dari laporan, dan chapter yang belum ada dibuat saat Terapkan.',
      'Periksa pratinjau, tekan Terapkan, lalu Lanjut ke langkah 2.',
      'Unggah Membership Dues Report, atau template jatuh tempo yang diunduh di langkah itu.',
      'Tekan Tinjau, Terapkan, lalu Selesai.',
    ],
  },
  {
    id: 'member-baru',
    judul: 'Tambah member dan tagih pendaftaran',
    ringkas: 'Untuk member baru yang tidak datang dari daftar visitor.',
    khususAdmin: true,
    langkah: [
      'Buka Member, lalu tekan Tambah Member. Isi nama dan chapter.',
      'Member masuk dengan status New Member.',
      'Buka detail member, lalu tekan Buat Invoice. Tipe Pendaftaran dan nama member sudah terpilih.',
      'Periksa nominal dan masa berlaku, lalu tekan Simpan sebagai Draft atau Buat & Kirim ke Paper.id.',
    ],
  },
  {
    id: 'ubah-member',
    judul: 'Ubah kontak atau status member',
    ringkas: 'Memperbaiki email dan no. HP, atau menandai member drop.',
    khususAdmin: true,
    langkah: [
      'Buka Member, lalu pilih nama member.',
      'Tekan Ubah Data, lalu perbaiki email atau no. HP.',
      'Untuk member drop, ubah Status keanggotaan ke Inactive.',
      'Tekan Simpan. Invoice yang sudah terbit tidak ikut berubah.',
    ],
  },
  {
    id: 'konfirmasi-renewal',
    judul: 'Konfirmasi renewal sampai invoice',
    ringkas: 'ST meminta konfirmasi, MC menjawab, ST meminta invoice, lalu finance menerbitkannya.',
    langkah: [
      'Buka Konfirmasi Renewal, pilih periode, lalu tekan Minta Konfirmasi.',
      'Pilih chapter, periksa member yang dicentang, lalu tekan Kirim permintaan. MC menerima email.',
      'MC membuka halaman yang sama, lalu menjawab Diterima, Tolak, atau Belum pasti.',
      'ST mencentang member yang Diterima, lalu tekan Minta invoice ke finance. Finance menerima email.',
      'Finance memilih status Invoice diminta, lalu tekan Buat Invoice (N). Kolom Status renewal menunjukkan tahap tiap member sampai lunas.',
    ],
  },
  {
    id: 'bukti-bayar',
    judul: 'Bukti bayar dan pengajuan ke BNI Global',
    ringkas: 'ST mengirim bukti, finance memverifikasi, lalu finance mengajukan renewal.',
    langkah: [
      'ST membuka detail invoice yang sudah terkirim, lalu tekan Kirim bukti bayar ke finance.',
      'Finance membuka Bukti Bayar, lalu tekan Setujui. Invoice menjadi lunas. Tolak dengan alasan bila buktinya salah.',
      'Setelah mengajukan renewal ke BNI Global, finance membuka Pengajuan BNI Global, mencentang membernya, lalu tekan Tandai sudah diajukan.',
    ],
  },
  {
    id: 'kirim-bayar',
    judul: 'Kirim invoice dan catat pembayaran',
    ringkas: 'Dari draft sampai lunas, termasuk pembayaran di luar Paper.id.',
    langkah: [
      'Buka Semua Invoice, centang draft yang akan dikirim, lalu tekan Kirim ke Paper.id. Tautan pembayaran dan jatuh tempo terisi otomatis.',
      'Pembayaran lewat Paper.id tercatat otomatis, dan due date member ikut maju.',
      'Untuk transfer bank atau tunai, buka detail invoice lalu tekan Catat Pembayaran Manual.',
    ],
  },
]

export interface FaqPanduan {
  tanya: string
  jawab: string
}

export const FAQ: { kelompok: string; butir: FaqPanduan[] }[] = [
  {
    kelompok: 'Data member dan chapter',
    butir: [
      {
        tanya: 'Bagaimana cara mengubah email atau no. HP member?',
        jawab: 'Buka Member, pilih nama member, lalu tekan Ubah Data. Email dan no. HP bisa diubah di sana. Khusus Admin.',
      },
      {
        tanya: 'Bagaimana cara mengubah status Active menjadi Inactive saat member drop?',
        jawab: 'Buka detail member, tekan Ubah Data, lalu ubah Status keanggotaan ke Inactive. Invoice yang sudah terbit tidak ikut berubah; batalkan dari halaman invoice bila perlu. Khusus Admin.',
      },
      {
        tanya: 'Bagaimana cara mengubah jatuh tempo membership?',
        jawab: 'Buka detail member, tekan Ubah Data, lalu isi Jatuh tempo membership. Renewal yang lunas memajukan tanggal ini otomatis, jadi pengubahan manual hanya untuk koreksi. Khusus Admin.',
      },
      {
        tanya: 'Bagaimana cara menghapus member?',
        jawab: 'Buka detail member, lalu tekan Hapus. Member yang sudah punya invoice tidak bisa dihapus; ubah statusnya ke Inactive supaya riwayat tagihannya utuh. Khusus Admin.',
      },
      {
        tanya: 'Bagaimana cara mengubah chapter?',
        jawab: 'Buka Chapter, lalu tekan ikon pensil di kartu chapter untuk mengubah nama tampil, area, dan kota. Khusus Admin.',
      },
      {
        tanya: 'Bagaimana cara menghapus chapter?',
        jawab: 'Buka Chapter, lalu tekan ikon tempat sampah di kartu chapter. Chapter hanya bisa dihapus bila tidak lagi punya member, invoice, atau akun ST/MC. Khusus Admin.',
      },
    ],
  },
  {
    kelompok: 'Konfirmasi renewal',
    butir: [
      {
        tanya: 'Bagaimana membatalkan permintaan konfirmasi yang salah kirim?',
        jawab: 'Di Konfirmasi Renewal, tekan Batalkan pada baris yang belum dijawab MC. Permintaan yang sudah dijawab tidak bisa dibatalkan, karena jawabannya menjadi dasar invoice renewal.',
      },
      {
        tanya: 'Bagaimana cara menambah periode renewal?',
        jawab: 'Tidak perlu ditambah. Dropdown periode di Konfirmasi Renewal otomatis berisi semester berjalan dan 5 semester berikutnya.',
      },
      {
        tanya: 'Apakah harus memilih chapter, atau otomatis ke chapter masing-masing?',
        jawab: 'Tidak harus memilih. Dengan Semua chapter, tiap permintaan otomatis masuk ke chapter member masing-masing, dan tiap MC menerima email untuk chapternya. Chapter hanya wajib dipilih bila memakai unggahan laporan Business Network International Connect.',
      },
      {
        tanya: 'Email konfirmasi tidak masuk ke MC. Apa yang harus diperiksa?',
        jawab: 'Lihat pesan setelah menekan Kirim permintaan: pesan itu menyebut jumlah email yang terkirim, yang gagal, dan chapter yang belum punya akun MC. Pastikan MC punya akun di chapter yang benar, lalu periksa folder spam. Untuk mengirim ulang, centang lagi member yang belum dijawab di Minta Konfirmasi.',
      },
    ],
  },
  {
    kelompok: 'Invoice',
    butir: [
      {
        tanya: 'Apakah Membership Fee otomatis berubah mengikuti KMK?',
        jawab: 'Ya. Sistem memeriksa kurs KMK setiap hari dan menghitung ulang biaya Rupiah, selama pilihan kurs otomatis di Biaya Keanggotaan menyala. Invoice yang sudah dibuat nominalnya tidak berubah.',
      },
      {
        tanya: 'Bagaimana membuat invoice pendaftaran untuk yang bukan dari visitor?',
        jawab: 'Pakai Tambah Member di halaman Member. Member masuk dengan status New Member, lalu tekan Buat Invoice di detail member untuk menagih pendaftaran. Member berstatus Active hanya bisa ditagih renewal.',
      },
      {
        tanya: 'Apakah invoice dari hasil konfirmasi renewal harus dibuat satu per satu?',
        jawab: 'Tidak. Di Konfirmasi Renewal, tekan Buat Invoice (N) untuk semua member yang dijawab Diterima oleh MC. Invoice terbuat sebagai draft, lalu centang di Semua Invoice untuk dikirim bersamaan.',
      },
      {
        tanya: 'Apakah bisa melihat invoice sudah diterima member?',
        jawab: 'Kartu Laporan Pengiriman di halaman Laporan menampilkan tanggal kirim, jumlah pengingat, dan status bayar tiap invoice. Paper.id hanya memberi kabar ke sistem saat invoice lunas, jadi sistem tidak tahu kapan invoice dibuka member.',
      },
      {
        tanya: 'Apakah link invoice yang sudah jatuh tempo masih bisa dipakai?',
        jawab: 'Pembayaran atas invoice yang lewat jatuh tempo tetap tercatat di sistem. Masa berlaku link ditentukan Paper.id. Bila link tidak bisa dibuka, kirim Pengingat dari detail invoice untuk menerbitkan tagihan baru.',
      },
      {
        tanya: 'Bagaimana bila invoice yang sudah dikirim dibatalkan?',
        jawab: 'Membatalkan invoice di sistem tidak mematikan link di Paper.id, dan pembayaran lewat link itu tidak tercatat di sistem. Batalkan juga invoice tersebut di dashboard Paper.id. Untuk banyak invoice Terlambat sekaligus, centang di Semua Invoice lalu tekan Batalkan Terlambat.',
      },
      {
        tanya: 'Bagaimana mengubah atau menghapus invoice?',
        jawab: 'Invoice draft bisa diubah nominal, masa berlaku, dan catatannya lewat Ubah Draft, atau dihapus lewat Hapus Draft. Invoice yang sudah dikirim tidak bisa dihapus; batalkan saja supaya riwayatnya utuh.',
      },
    ],
  },
  {
    kelompok: 'Laporan dan saldo',
    butir: [
      {
        tanya: 'Apakah due date member otomatis diperbarui?',
        jawab: 'Ya. Saat invoice lunas, lewat Paper.id maupun catat manual, due date member otomatis maju ke akhir masa invoice dan tidak pernah mundur.',
      },
      {
        tanya: 'Bagaimana cara menarik dana (withdraw)?',
        jawab: 'Platform ini tidak memiliki menu saldo atau penarikan. Dana pembayaran masuk ke akun Paper.id, jadi saldo dan penarikan diurus di sana.',
      },
    ],
  },
]
