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

## Putaran 19 November 2026

Sumber: lembar feedback klien dan penjelasan lanjutan lewat pesan pada hari yang sama.

Butir berstatus `selesai` di putaran ini masuk lewat [PR #112](https://github.com/syabanf/bni-finance/pull/112).

### Ringkasan

| No | Masukan | Status |
| --- | --- | --- |
| F-01 | Mengubah atau menambah keterangan produk di invoice | `sudah ada` |
| F-02 | Mengubah atau menambah keterangan deskripsi di invoice | `sudah ada` |
| F-03 | Detail rekening bila member ingin transfer | `selesai` |
| F-04 | Syarat dan ketentuan di invoice | `selesai` |
| F-05 | Hapus invoice yang overdue | `tidak dikerjakan` |
| F-06 | Apakah invoice mengikuti format Paper.id | `sudah ada` |
| F-07 | Rincian pengaturan biaya per komponen | `dikerjakan` |
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

**F-07. Rincian biaya per komponen.** `dikerjakan`

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

Sebabnya: daftar itu hanya memuat enam semester dari semester berjalan. Sekarang pilihan
terakhir di daftar, **Tampilkan periode berikutnya…**, menambahkan empat semester setiap
kali dipilih, tanpa memindahkan periode yang sedang dilihat.

**F-10. Kolom Ditugaskan menampilkan ID.** `selesai`

Kolom itu menampilkan deretan huruf dan angka, yaitu id akun MC, bila namanya tidak
ditemukan. Halaman mencari nama itu dari daftar akun yang dimuat terpisah. Daftar itu hanya
bisa dibaca admin, dan bila gagal dimuat, yang tersisa untuk ditampilkan hanya id-nya.

Sekarang server mengirim nama MC bersama tiap baris, jadi namanya tampil untuk admin, ST,
maupun MC tanpa bergantung pada daftar akun. Id akun tidak ditampilkan lagi dalam keadaan
apa pun.
