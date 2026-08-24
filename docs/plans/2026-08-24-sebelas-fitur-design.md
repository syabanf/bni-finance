# Desain — sebelas fitur BNI Finance Hub

**Tanggal**: 2026-08-24
**Status**: disetujui, belum dikerjakan

Sebelas permintaan fitur, dikelompokkan menjadi lima alur kerja. Dokumen ini
mencatat keputusan desainnya beserta alasannya — terutama alasan menolak
alternatif yang kelihatan lebih mudah.

---

## Keadaan saat ini

Yang sudah ada dan akan dipakai ulang:

| Hal | Keadaan |
|---|---|
| Peran | hanya `admin` (semua) dan `user` (baca), **tanpa lingkup chapter** |
| Status invoice | `draft`, `sent`, `paid`, `overdue`, `cancelled` |
| Status member | `active`, `inactive`, `pending` |
| Sync BNI VM | `POST /api/v1/sync` — chapter & member, sudah jalan |
| Reminder | manual per invoice, `POST /api/v1/invoices/{id}/remind` |
| Unggah berkas | `POST /api/v1/uploads` → `UPLOAD_DIR` |
| Notifikasi | `paperid_send_email`, `paperid_send_whatsapp` di `app_settings` |
| Blackbox | merekam seluruh panggilan masuk & keluar |

---

## A. Peran dan lingkup chapter

Fondasi. B, C, dan D semuanya bergantung padanya, jadi dikerjakan pertama.

`users` mendapat kolom `chapter_id` (null untuk peran nasional). Enum `user_role`
bertambah `st` dan `mc`.

| Peran | Lingkup | Boleh |
|---|---|---|
| `admin` | nasional | semuanya |
| `st` | satu chapter | baca + tulis di chapternya |
| `mc` | satu chapter | baca + menjawab konfirmasi renewal |
| `user` | nasional | baca saja |

### Di mana lingkup ditegakkan

**Postgres tidak bisa menegakkannya.** Backend menyambung sebagai SATU peran
tepercaya, jadi database melihat satu identitas saja dan tidak punya apa pun
untuk dipakai sebagai kunci kebijakan per-baris. Ini sudah berlaku untuk seluruh
otorisasi yang ada dan tidak berubah di sini.

**Keputusan: filter di repository, lingkup dibawa lewat request context.**

Middleware menaruh `chapterID` ke context; setiap `List` dan `GetByID` menambahkan
`where chapter_id = $n` bila lingkupnya terisi.

Alternatif yang ditolak — memeriksa peran di service layer per endpoint. Lebih
mudah dibaca, tapi ada 30+ endpoint dan yang terlewat **tidak gagal**: ia
mengembalikan data chapter lain dengan status 200. Kegagalan yang sunyi seperti
itu persis kelas bug yang sudah beberapa kali menggigit proyek ini. Dengan filter
di repository, yang terlewat justru tampak sebagai baris yang hilang — keras,
tidak diam.

### Cara membuktikannya

Tes batas akses mengikuti `internal/api/authboundary_test.go` yang sudah ada:
setiap endpoint diuji dengan token ST chapter A yang menuntut data chapter B.
Tes ini satu-satunya bukti bahwa batasnya benar-benar ada, karena tidak ada
lapisan lain yang menegakkannya.

---

## B. Alur konfirmasi renewal

Tabel baru `renewal_requests`:

```
member_id, chapter_id, period,
requested_by (ST), assigned_mc,
answer  -- pending | will_renew | will_not | unsure
answered_at, note
```

Alurnya:

1. ST membuka daftar member yang renewal-nya jatuh tempo, menekan
   "Minta konfirmasi" — satu baris `pending` per member, ter-tag ke MC chapter itu
2. MC melihat daftar tugasnya, menjawab per member
3. ST menerbitkan invoice hanya untuk yang `will_renew` — langsung menyambung ke
   bulk send di alur C

Satu baris per member per periode, supaya permintaan yang sama tidak menumpuk
saat ST menekan tombolnya dua kali.

---

## C. Pengiriman

### Bulk send dengan filter chapter

Filter per chapter dan status, lalu **looping panggilan API** — bukan satu
permintaan besar.

Tahan gagal: satu invoice gagal tidak menghentikan sisanya. Tiap hasil masuk
blackbox, dan ringkasan akhir menyebut "berhasil N, gagal M" beserta alasan per
baris. Pengiriman massal yang berhenti di tengah tanpa memberitahu apa yang sudah
terkirim adalah pengiriman yang tidak bisa diulang dengan aman — nomor Paper.id
terbakar permanen.

### Faktur Pajak sebagai lampiran

Kolom `tax_invoice_file` di `invoices`, diunggah lewat `POST /api/v1/uploads`
yang sudah ada. Sistem tidak membuat isinya.

**RISIKO YANG BELUM TERPECAHKAN.** `CreateInput` pada `internal/paperid/client.go`
tidak punya field lampiran, dan dokumentasi Paper.id yang tersedia tidak
menyebutkan attachment pada `store-invoice`. Jadi belum ada yang membuktikan
lampiran bisa dikirim lewat Paper.id sama sekali.

Karena itu fitur ini dikerjakan **terakhir**, dan didahului pemastian ke API-nya.
Kalau ternyata tidak didukung, dua jalan keluar: menyertakan URL berkas di catatan
invoice, atau mengirim lampirannya lewat kanal kita sendiri. Yang tidak boleh
terjadi adalah menuliskan fiturnya lalu berharap.

---

## D. Reminder dan notifikasi

### Worker

Goroutine ticker di dalam binary Go yang sudah ada, bukan kontainer terpisah.
Docker sudah menjalankan satu proses; menambah kontainer berarti menambah satu
hal lagi yang bisa mati diam-diam tanpa ada yang menyadari.

Dijaga **advisory lock Postgres**, supaya dua replika tidak mengirim dobel.

### Jadwal H-N

Pengaturan `reminder_offsets` berisi daftar hari, misalnya `[7, 3, 1]`.

Tiap pasangan invoice × offset dikirim **sekali saja**, dijaga tabel
`reminder_log`. Tanpa penjaga itu, worker yang restart akan mengirim ulang
seluruh pengingat yang sudah pernah dikirim — dan tiap pengiriman Paper.id
membakar nomor secara permanen.

### On/off notifikasi

Memperluas `paperid_send_email` dan `paperid_send_whatsapp` yang sudah ada,
ditambah sakelar induk `notifications_enabled` yang mematikan seluruhnya.

---

## E. Data dan status

### Import chapter & member

Sync BNI VM sudah ada dan tetap dipakai. Yang ditambahkan: unggah **CSV/XLSX**.

**Pratinjau sebelum menulis** — berapa baris baru, berapa diperbarui, baris mana
yang ditolak dan kenapa. Impor yang langsung menulis tanpa pratinjau merusak data
secara diam-diam, dan pada data keanggotaan kerusakan itu baru ketahuan saat
tagihan salah kirim.

### Status Termination

Nilai baru pada enum `invoice_status`, plus `audit_action` yang sepadan.
Idempoten, mengikuti pola `alter type … add value 'reminded'` yang sudah ada di
`db/init.sql`.

Melekat pada **invoice**, bukan member — membedakan "dibatalkan biasa" dari
"dibatalkan karena keanggotaan diputus".

### Denda keterlambatan

**Hanya ditampilkan, tidak ditagih otomatis.**

Pengaturan `denda_per_hari` dan `denda_satuan`. Dihitung saat dibaca:
`hari telat × nominal`. Tidak mengubah invoice, tidak menyentuh Paper.id, tidak
ada penulisan berkala.

Ini keputusan yang membuat fiturnya kecil. Denda yang menempel dan tumbuh di
invoice akan memaksa nominal berubah seiring waktu, dan setiap perubahan nominal
pada invoice yang sudah terkirim ke Paper.id berarti tagihan yang tidak lagi
cocok dengan yang diterima member.

---

## Urutan pengerjaan

1. **A** — peran dan lingkup chapter (fondasi)
2. **E** — import, termination, denda (paling kecil, cepat terlihat)
3. **B** — alur konfirmasi renewal
4. **D** — worker, jadwal, sakelar notifikasi
5. **C** — bulk send, lalu Faktur Pajak setelah API Paper.id dipastikan

Beberapa PR terpisah, bukan satu.
