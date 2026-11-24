# Feedback Klien

Catatan setiap masukan dari klien, jawabannya, dan statusnya. Dokumen ini dipakai untuk
tiga hal: menjawab klien dengan kalimat yang sama setiap kali, melihat apa yang masih
menunggu keputusan, dan menelusuri masukan mana yang menghasilkan perubahan apa.

## Cara memakai

1. Masukan baru masuk sebagai **putaran** baru di bagian atas, dengan tanggal diterimanya.
2. Setiap butir mendapat nomor (`F-03`), satu status, dan jawaban yang bisa disalin ke klien.
3. Butir yang butuh keputusan menuliskan pertanyaannya dan pilihan yang tersedia.
4. Saat butir selesai, tautkan PR-nya dan ubah statusnya. Butirnya tidak dihapus.

| Status | Arti |
| --- | --- |
| `sudah ada` | Fiturnya sudah tersedia. Yang dibutuhkan klien adalah petunjuk tempatnya |
| `selesai` | Dikerjakan karena masukan ini, sudah di `main` |
| `dikerjakan` | Sedang dikerjakan, ada PR terbuka |
| `menunggu keputusan` | Tidak bisa dikerjakan sebelum pertanyaannya dijawab |
| `tidak dikerjakan` | Sengaja tidak dibuat, dengan alasannya |

---

## Putaran 24 November 2026

Sumber: pesan klien berisi alur renewal membership, dan pertanyaan lanjutan apakah halaman
Konfirmasi Renewal bisa menampilkan status sampai lunas.

Kedua butir masuk lewat [PR #117](https://github.com/syabanf/bni-finance/pull/117).

### Ringkasan

| No | Masukan | Status |
| --- | --- | --- |
| F-11 | Alur renewal enam langkah, dari email BNI Global sampai finance mengajukan renewal | `selesai` |
| F-12 | Status lunas atau belum ditampilkan di Konfirmasi Renewal | `selesai` |

**F-11. Alur renewal.** `selesai`

Alur dari klien:

1. BNI Global mengirim email ke MC dan ST berisi member yang akan jatuh tempo.
2. ST meminta daftar member yang sudah disetujui MC.
3. ST meminta invoice ke finance national office.
4. Invoice dikirim ke member berdasarkan permintaan ST.
5. ST memberikan bukti bayar ke finance.
6. Finance mengajukan renewal membership.

Langkah 1 terjadi di luar sistem, dan langkah 2 sudah ada lewat Konfirmasi Renewal.
Perubahan untuk langkah 3 sampai 6:

- **Langkah 3.** ST mencentang member yang sudah Diterima MC lalu menekan **Minta invoice
  ke finance**. Finance menerima email dan melihat antreannya di Konfirmasi Renewal dengan
  saringan status **Invoice diminta**.
- **Langkah 4.** Hanya finance (admin) yang membuat dan mengirim invoice. ST tidak lagi
  punya tombol buat, kirim, atau tandai lunas.
- **Langkah 5.** ST mengirim bukti transfer dari halaman detail invoice. Bukti itu berstatus
  **Menunggu verifikasi** sampai finance menyetujuinya di halaman **Bukti Bayar**, dan baru
  saat itu invoice menjadi lunas. Bila ditolak, finance menulis alasannya dan ST bisa mengirim
  bukti baru. Pembayaran lewat Paper.id tetap lunas otomatis.
- **Langkah 6.** Halaman **Pengajuan BNI Global** memuat invoice renewal yang sudah lunas.
  Finance mencentang yang sudah diajukan, mengisi tanggalnya, lalu menandainya. Tab
  **Belum diajukan** menjadi daftar kerja supaya tidak ada member yang terlewat.

Kedua halaman baru punya pencarian nomor invoice atau nama member dan saringan chapter.

Jawaban untuk klien: alur itu sudah diterapkan. ST meminta invoice dan mengirim bukti bayar,
finance menerbitkan invoice, memverifikasi bukti, dan menandai renewal yang sudah diajukan
ke BNI Global.

**F-12. Status sampai lunas di Konfirmasi Renewal.** `selesai`

Kolom **Status renewal** menampilkan tahap terakhir tiap member: menunggu jawaban MC, siap
diminta invoice, invoice diminta, belum dibayar, bukti menunggu finance, lunas, dan diajukan
ke BNI Global. Nomor invoicenya tertaut ke halaman detail. Saringan **Semua status** memilih
satu tahap, dan pilihan periode **Semua periode** membuka antrean lintas periode.

Jawaban untuk klien: bisa. Halaman Konfirmasi Renewal menampilkan status tiap member sampai
lunas dan sampai diajukan ke BNI Global, tanpa perlu membuka halaman invoice.

---

## Putaran 19 November 2026

Sumber: lembar feedback klien dan penjelasan lanjutan lewat pesan pada hari yang sama.

Butir berstatus `selesai` di putaran ini masuk lewat [PR #112](https://github.com/syabanf/bni-finance/pull/112),
kecuali F-07 yang masuk lewat [PR #114](https://github.com/syabanf/bni-finance/pull/114) dan F-09
yang diganti lewat [PR #116](https://github.com/syabanf/bni-finance/pull/116).

### Ringkasan

| No | Masukan | Status |
| --- | --- | --- |
| F-01 | Mengubah atau menambah keterangan produk di invoice | `sudah ada` |
| F-02 | Mengubah atau menambah keterangan deskripsi di invoice | `sudah ada` |
| F-03 | Detail rekening bila member ingin transfer | `selesai` |
| F-04 | Syarat dan ketentuan di invoice | `selesai` |
| F-05 | Hapus invoice yang overdue | `tidak dikerjakan` |
| F-06 | Apakah invoice mengikuti format Paper.id | `sudah ada` |
| F-07 | Rincian pengaturan biaya per komponen | `selesai` |
| F-08 | Bila MC tidak menyetujui renewal, apakah memilih "dibatalkan" | `sudah ada` |
| F-09 | Menambah periode di Konfirmasi Renewal | `selesai` |
| F-10 | Kolom Ditugaskan menampilkan ID, seharusnya nama MC | `selesai` |

### Invoice

**F-01 dan F-02. Keterangan produk dan deskripsi.** `sudah ada`

Keduanya diatur di **Pengaturan → Biaya Keanggotaan → Teks Invoice di Paper.id**, terpisah
untuk Pendaftaran dan Renewal. Perubahan berlaku untuk invoice yang dikirim setelah
disimpan. Invoice yang sudah terkirim tidak bisa diubah teksnya, karena nomornya sudah
terbit di Paper.id.

Perbaikan yang ikut masuk: pratinjau dan unduhan invoice di aplikasi dulu menulis nama
produknya sendiri, jadi teks yang diubah admin tampil di invoice Paper.id tetapi tidak di
salinan yang diunduh pengurus. Sekarang keduanya membaca pengaturan yang sama.

**F-03. Detail rekening untuk transfer.** `selesai`

Kolom baru **Catatan invoice** di kartu yang sama. Isinya dikirim ke kolom `notes` invoice
Paper.id dan ikut tercetak di invoice, pengingat, pratinjau, dan unduhan.

Yang perlu disampaikan ke klien: pembayaran lewat transfer langsung tidak melewati
Paper.id, jadi invoicenya tidak lunas otomatis. Pengurus mencatatnya lewat tombol
**Catat Pembayaran Manual** di halaman detail invoice.

**F-04. Syarat dan ketentuan.** `selesai`

Kolom baru **Syarat dan ketentuan** di kartu yang sama, dikirim ke kolom `terms_condition`
invoice Paper.id. Bila dikosongkan, bagian itu tidak tampil di invoice Paper.id.

**F-05. Hapus invoice yang overdue.** `tidak dikerjakan`

Keputusan 19 November 2026: **Batalkan sudah cukup.** Penghapusan invoice terkirim tidak
dibuat.

Hanya invoice **draft** yang bisa dihapus. Invoice yang sudah terkirim, termasuk yang
overdue, bisa **Dibatalkan** dengan alasan, atau **Diputus** bila keanggotaannya berakhir.
Keduanya tetap tersimpan sebagai riwayat.

Alasannya: invoice yang sudah terkirim punya tautan pembayaran Paper.id yang tetap hidup.
Kalau barisnya dihapus dari sistem lalu member membayar lewat tautan itu, uangnya masuk
tanpa invoice yang bisa dicocokkan. Invoice yang dibatalkan juga keluar dari hitungan
ditagih dan outstanding di Laporan, jadi untuk angka, membatalkan sama dengan menghapus.

**F-06. Apakah mengikuti format Paper.id.** `sudah ada`

Ya. Invoice yang diterima member dicetak oleh Paper.id dengan tata letaknya. Sistem ini
mengisi kolom yang disediakan Paper.id:

| Kolom Paper.id | Diisi dari |
| --- | --- |
| Produk dan deskripsi | Pengaturan (F-01, F-02) |
| Catatan (`notes`) | Pengaturan (F-03) |
| Syarat dan ketentuan (`terms_condition`) | Pengaturan (F-04) |
| Baris item | Satu baris per komponen harga bila rincian aktif (F-07). Tanpa rincian, satu baris berisi total |
| Pajak per baris (`tax_id`) | Belum dipakai. Butuh pajak yang didaftarkan di dasbor Paper.id |

Logo, warna, dan tata letak invoice diatur di dasbor Paper.id, bukan dari sistem ini.

### Pengaturan biaya

**F-07. Rincian biaya per komponen.** `selesai`

Rincian yang diminta klien:

| Komponen | Pendaftaran | Renewal |
| --- | --- | --- |
| Registration Fee | $180,00 | tidak ada |
| Membership Fee | $648,83 | $648,83 |
| VAT 11% | $91,17 | $71,37 |
| Online Training Investment Bundling | Rp 500.000 | tidak ada |
| **Jumlah** | **$920,00 + Rp 500.000** | **$720,20** |

Keputusan 19 November 2026, mengikuti usulan:

| | Pertanyaan | Keputusan |
| --- | --- | --- |
| a | Baris terpisah atau satu baris total di invoice member? | Baris terpisah, satu per komponen |
| b | Apakah biaya training kena VAT? | Tidak |
| c | Kapan harga baru berlaku? | Saat admin menyimpan rincian di Pengaturan. Deploy tidak mengubah harga |
| d | Pembulatan ke Rupiah | Tiap baris dibulatkan, lalu dijumlah |

Yang dibuat:

1. **Pengaturan → Biaya Keanggotaan → Nominal Biaya** punya bagian **Rinci harga per
   komponen**. Admin mengisi Registration Fee, Membership Fee, persen VAT, dan biaya
   training, lalu melihat baris dan totalnya sebelum menyimpan.
2. VAT dihitung dari komponen Dollar sebelum pajak dan dibulatkan ke sen, sama dengan
   lembar klien. Tiap baris lalu dikalikan kurs dan dibulatkan ke rupiah. Harga per tipe
   adalah jumlah baris-barisnya.
3. Sebelum Simpan ditekan, halaman menyebut harga lama dan harga baru bila keduanya berbeda.
4. Tiap invoice menyimpan rinciannya sendiri saat dibuat. Perubahan harga atau kurs
   sesudahnya tidak mengubah invoice yang sudah ada.
5. Invoice dikirim ke Paper.id sebagai satu baris per komponen. Pratinjau, unduhan, dan
   halaman detail invoice menampilkan baris yang sama.

Diuji ke Paper.id staging pada 19 November 2026 dengan satu invoice pendaftaran. Paper.id
menerima keempat baris, dan invoice PDF-nya menampilkan tiap baris, total yang sama dengan
jumlah barisnya, catatan tiga baris di bagian Keterangan, dan Syarat & Ketentuan.

Dua hal yang terlihat di invoice Paper.id itu dan belum diputuskan:

- **Baris "Pajak" di bagian total berisi Rp 0.** VAT dikirim sebagai baris item biasa, jadi
  kolom pajak milik Paper.id tidak terisi. Supaya VAT tampil di kolom itu, pajaknya harus
  didaftarkan di dasbor Paper.id dan id-nya dikirim sebagai `tax_id` per baris.
- **Deskripsi tiap baris sama.** Keempat baris memakai deskripsi produk tipe invoicenya,
  termasuk baris VAT dan training.

Tiga hal yang perlu disampaikan ke klien:

- **Rincian belum aktif setelah deploy.** Kolomnya sudah terisi angka dari lembar klien,
  tetapi harga tetap seperti sekarang sampai admin mencentang rincian dan menyimpannya.
- **Mengaktifkannya mengubah harga.** Pada kurs 16.000, pendaftaran menjadi Rp 15.220.000
  dan renewal Rp 11.523.200. Angka pastinya mengikuti kurs yang berlaku saat disimpan.
- **Nominal yang diketik sendiri tetap satu baris.** Invoice yang nominalnya berbeda dari
  harga berlaku tidak punya rincian, karena baris pengaturan tidak berjumlah sama dengannya.

### Konfirmasi renewal

**F-08. Bila MC tidak menyetujui.** `sudah ada`

MC memilih **Tolak** pada baris member itu. Tidak ada yang perlu dibatalkan: invoice
renewal memang tidak bisa diterbitkan untuk member yang jawabannya Tolak, jadi belum ada
invoice yang terbentuk.

| Jawaban MC | Akibatnya |
| --- | --- |
| Diterima | Invoice renewal boleh diterbitkan, satu kali untuk konfirmasi itu |
| Tolak | Invoice renewal tidak bisa diterbitkan |
| Belum pasti | Invoice renewal belum bisa diterbitkan, dan jawabannya bisa diubah nanti |

Bila invoice renewal sudah terlanjur terbit sebelum MC menjawab Tolak, invoice itu
dibatalkan dari halaman detailnya (lihat F-05).

**F-09. Menambah periode.** `selesai`

Penjelasan klien: daftar periode berhenti di 2029 · Jan–Jun, dan periode sesudahnya tidak
bisa dipilih.

Sebabnya: daftar itu hanya memuat enam semester dari semester berjalan.

Perbaikan pertama di PR #112 menambah pilihan **Tampilkan periode berikutnya…** di
daftar itu. Pada 20 November 2026 klien mengulang pertanyaannya, dan perbaikan itu diganti
dengan master data periode di [PR #116](https://github.com/syabanf/bni-finance/pull/116):

- Admin membuka **Konfirmasi Renewal → Kelola Periode**. Formulirnya sudah terisi periode
  sesudah periode terakhir, jadi menambah 2029 · Jul–Des cukup satu klik **Tambah periode**.
- Periode bisa diberi nama sendiri dan rentang tanggal bebas, misalnya per kuartal.
- Periode bisa diubah nama dan tanggalnya kapan saja. Periode yang sudah punya permintaan
  konfirmasi tidak bisa dihapus.
- Halaman menolak nama yang sudah dipakai dan rentang yang beririsan sebelum disimpan.
- ST dan MC melihat daftar yang sama, tanpa tombol Kelola Periode.

Jawaban untuk klien: bisa. Admin menambah periode 2029 · Jul–Des lewat tombol Kelola
Periode di halaman Konfirmasi Renewal, dan periode itu langsung bisa dipilih semua chapter.

**F-10. Kolom Ditugaskan menampilkan ID.** `selesai`

Kolom itu menampilkan deretan huruf dan angka, yaitu id akun MC, bila namanya tidak
ditemukan. Halaman mencari nama itu dari daftar akun yang dimuat terpisah. Daftar itu hanya
bisa dibaca admin, dan bila gagal dimuat, yang tersisa untuk ditampilkan hanya id-nya.

Sekarang server mengirim nama MC bersama tiap baris, jadi namanya tampil untuk admin, ST,
maupun MC tanpa bergantung pada daftar akun. Id akun tidak ditampilkan lagi dalam keadaan
apa pun.
