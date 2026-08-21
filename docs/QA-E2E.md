# Skenario QA End-to-End — BNI Finance Hub

61 skenario di 12 modul. Diturunkan dari kode yang benar-benar ada: rute di
`src/app/router.tsx`, endpoint yang terdaftar di mux, dan aturan transisi di
`backend/internal/domain/invoice.go` — bukan daftar generik.

Berkas ini adalah SUMBER. Versi Excel untuk eksekusi QA dihasilkan darinya:

```bash
python3 scripts/build-qa-xlsx.py
```

Hasilnya `test-report/skenario-qa-e2e.xlsx` (gitignored) dengan kolom
Status / Tester / Tanggal / Temuan siap diisi, dropdown status, dan autofilter.

## Prioritas

| Kode | Arti |
| --- | --- |
| P1 | Menghalangi rilis. Harus hijau sebelum ship. |
| P2 | Penting; boleh menyusul dengan catatan. |
| P3 | Pelengkap. |

Jenis: Happy path · Negatif · Keamanan · Balapan (concurrency) · Integrasi · Beban.

## Prasyarat lingkungan

- `backend/.env` terisi (`npm run setup`)
- Database disiapkan dengan satu perintah: `cd backend && make db-reset`
- Kredensial Paper.id staging terpasang untuk skenario Integrasi

`db/init.sql` sudah membuat dua akun, dan emailnya sama persis dengan akun demo
pada mode Data Contoh — jadi skenario yang sama berlaku di kedua mode:

| Peran | Email | Kata sandi |
| --- | --- | --- |
| Admin | `admin@bni-finance.com` | `admin123` |
| User | `user@bni-finance.com` | `user123` |

> **Kredensial pengembangan.** Kata sandi di atas tertulis di dalam repo dan
> sama di setiap salinan. Jangan pernah menjalankan `db/init.sql` pada database
> produksi yang masih kosong; produksi memakai `SEED_ADMIN_*` dari environment.

> **Kanal pengiriman.** Skenario Integrasi yang mengirim email/WhatsApp akan
> menghubungi kontak SUNGGUHAN di data. Pada lingkungan uji, arahkan seluruh
> kontak ke milik tim penguji lebih dulu, atau matikan kanalnya.

> **Nomor invoice tidak bisa didaur ulang.** Paper.id membakar nomor invoice
> secara permanen begitu dipakai. Membersihkan database lokal tidak
> mengembalikannya.

> **Pembayaran mandiri (Xendit) dihapus.** Seluruh permukaan bayar publik —
> `/pay/:id`, endpoint publik, webhook Xendit, dan halaman Metode Pembayaran —
> sudah tidak ada. Member membayar lewat tautan Paper.id. Skenario PAY-03
> sampai PAY-07 dan SET-03 dihapus bersamanya.

## Penjaga regresi

Sembilan skenario memuat catatan bug yang PERNAH benar-benar terjadi di proyek
ini. Itu bukan hiasan — skenario tersebut adalah penjaga regresi dan paling
layak dijalankan lebih dulu: `ROLE-06`, `INV-02`, `INV-06`, `MEM-03`, `INV-07`,
`SET-01`, `AUTH-06`, `X-02`, `RSP-04`.

## Rekap

| Modul | Total | P1 | P2 | P3 |
| --- | ---: | ---: | ---: | ---: |
| Autentikasi | 8 | 4 | 4 | 0 |
| Peran & Akses | 6 | 6 | 0 | 0 |
| Member | 5 | 2 | 3 | 0 |
| Chapter | 2 | 1 | 1 | 0 |
| Invoice | 12 | 7 | 4 | 1 |
| Pembayaran | 3 | 2 | 1 | 0 |
| Dashboard & Laporan | 4 | 1 | 3 | 0 |
| Pengaturan | 4 | 2 | 2 | 0 |
| Alat Admin | 6 | 0 | 4 | 2 |
| Sumber Data | 3 | 3 | 0 | 0 |
| Responsif | 4 | 1 | 2 | 1 |
| Lintas Fungsi | 4 | 1 | 2 | 1 |
| **TOTAL** | **61** | **30** | **26** | **5** |

## Skenario

Sel multi-baris memakai `<br>`. Kolom dibaca apa adanya oleh
`scripts/build-qa-xlsx.py`, jadi jaga jumlah dan urutan kolomnya.

| ID | Modul | Pri | Judul | Prasyarat | Langkah | Hasil yang diharapkan | Jenis |
| --- | --- | --- | --- | --- | --- | --- | --- |
| AUTH-01 | Autentikasi | P1 | Masuk sebagai admin dengan kredensial benar | Mode Backend API aktif. Akun admin ada di database. | 1. Buka /login<br>2. Isi email & kata sandi admin<br>3. Klik Masuk | Diarahkan ke /dashboard. Nama pengguna tampil di header. Menu admin (Pengaturan, Blackbox, API Console) terlihat di sidebar. | Happy path |
| AUTH-02 | Autentikasi | P1 | Masuk ditolak dengan kata sandi salah | Akun admin ada. | 1. Buka /login<br>2. Isi email benar, kata sandi salah<br>3. Klik Masuk | Tetap di /login. Muncul pesan galat. TIDAK menyebutkan apakah emailnya yang salah atau sandinya — supaya email tidak bisa dipanen. | Negatif |
| AUTH-03 | Autentikasi | P1 | Halaman terlindungi menolak pengunjung tanpa token | Belum masuk (localStorage bersih). | 1. Buka langsung /dashboard, /invoices, /members, /reports satu per satu | Setiap kali diarahkan ke /login. Tidak ada data yang sempat terlihat sekilas. | Keamanan |
| AUTH-04 | Autentikasi | P2 | Token kedaluwarsa memaksa masuk ulang | Sudah masuk. TOKEN_TTL disetel pendek pada backend uji. | 1. Masuk<br>2. Tunggu melewati TTL<br>3. Lakukan aksi yang memanggil API (misal buka /invoices) | Diarahkan ke /login, bukan menampilkan halaman kosong atau galat mentah. | Negatif |
| AUTH-05 | Autentikasi | P1 | Keluar membersihkan sesi sepenuhnya | Sudah masuk. | 1. Klik Keluar<br>2. Tekan tombol Back peramban<br>3. Periksa localStorage | Kembali ke /login, tidak bisa menembus lewat Back. Token tidak tersisa di localStorage. | Keamanan |
| AUTH-06 | Autentikasi | P2 | Ganti kata sandi sendiri | Sudah masuk sebagai user mana pun. | 1. Buka /profile<br>2. Isi kata sandi lama & baru<br>3. Simpan<br>4. Keluar, masuk dengan kata sandi baru | Tersimpan tanpa galat. Kata sandi lama ditolak, kata sandi baru diterima.<br>CATATAN: ini jalur PUT — dulu pernah gagal karena PUT tidak ada di CORS preflight. | Happy path |
| AUTH-07 | Autentikasi | P2 | Quick login mati secara bawaan | AUTH_QUICK_LOGIN tidak diset pada backend. | 1. Buka /login di mode Backend API<br>2. Amati area "masuk cepat"<br>3. Panggil GET /api/v1/auth/quick-login | Tidak ada tombol masuk cepat. Endpoint membalas 404 (bukan 403) — tidak memberi petunjuk bahwa rutenya ada. | Keamanan |
| AUTH-08 | Autentikasi | P2 | Quick login menolak akun di luar daftar izin | AUTH_QUICK_LOGIN berisi satu email saja. | 1. POST /api/v1/auth/quick-login dengan email admin LAIN yang tidak terdaftar | HTTP 403. Tidak ada token yang diterbitkan. | Keamanan |
| ROLE-01 | Peran & Akses | P1 | User biasa tidak melihat menu admin | Masuk sebagai peran user. | 1. Amati sidebar<br>2. Coba buka langsung /settings, /blackbox, /api-console, /invoices/new | Menu admin tidak tampil. Akses langsung lewat URL ditolak/dialihkan, bukan menampilkan halaman lalu gagal saat menyimpan. | Keamanan |
| ROLE-02 | Peran & Akses | P1 | User biasa bisa membaca dan mengekspor | Masuk sebagai peran user. | 1. Buka /invoices, /members, /payments, /reports<br>2. Jalankan ekspor pada /reports | Semua data tampil. Ekspor berhasil diunduh. Tidak ada tombol ubah/hapus. | Happy path |
| ROLE-03 | Peran & Akses | P1 | User biasa ditolak saat menulis lewat API | Token peran user. | 1. POST /api/v1/invoices dengan token user<br>2. PATCH /api/v1/members/{id}<br>3. DELETE /api/v1/chapters/{id} | Ketiganya HTTP 403. Penjagaan ada di server, bukan sekadar menyembunyikan tombol. | Keamanan |
| ROLE-04 | Peran & Akses | P1 | Admin terakhir tidak bisa diturunkan perannya | Hanya ada SATU admin di sistem. | 1. Buka pengelolaan pengguna<br>2. Ubah peran admin terakhir menjadi user | HTTP 409 dan pesan jelas. Sistem TIDAK boleh berakhir tanpa admin. | Negatif |
| ROLE-05 | Peran & Akses | P1 | Admin terakhir tidak bisa dihapus | Hanya ada SATU admin. | 1. DELETE /api/v1/users/{id} untuk admin terakhir | HTTP 409. Akun tetap ada. | Negatif |
| ROLE-06 | Peran & Akses | P1 | Dua penurunan peran serentak tidak menghabiskan admin | Tepat DUA admin di sistem. | 1. Kirim dua PATCH peran→user bersamaan untuk kedua admin (dua tab / dua curl paralel) | Tepat satu berhasil, satu 409. Jumlah admin tersisa minimal 1.<br>CATATAN: bug nyata — 3 dari 3 percobaan menyisakan NOL admin. | Balapan |
| MEM-01 | Member | P1 | Membuat member baru | Masuk sebagai admin. Minimal satu chapter ada. | 1. Buka /members<br>2. Tambah member: nama, email, telepon, chapter, status aktif<br>3. Simpan | Member muncul di daftar. Detail di /members/{id} sesuai isian. | Happy path |
| MEM-02 | Member | P2 | Validasi email dan telepon | Form tambah member terbuka. | 1. Isi email tanpa @<br>2. Isi telepon dengan huruf<br>3. Simpan | Ditolak dengan pesan per-field. Tidak ada baris tersimpan sebagian. | Negatif |
| MEM-03 | Member | P1 | Mengubah kontak member | Member sudah ada dan pernah ditagih. | 1. Buka /members/{id}<br>2. Ubah nomor telepon<br>3. Simpan<br>4. Terbitkan invoice baru untuk member itu<br>5. Kirim ke Paper.id | Perubahan tersimpan DAN pengiriman berikutnya tetap berhasil.<br>CATATAN: customer.id di Paper.id terikat ke detail kontak; dulu ini menyebabkan galat permanen "Failed partner doesn't match". | Integrasi |
| MEM-04 | Member | P2 | Menonaktifkan member | Member aktif dengan invoice berjalan. | 1. Ubah status member menjadi tidak aktif<br>2. Periksa daftar jatuh tempo perpanjangan | Member tidak lagi muncul sebagai kandidat perpanjangan. Invoice lama tetap utuh dan bisa dibayar. | Happy path |
| MEM-05 | Member | P2 | Filter dan cari member | Beberapa member dengan chapter dan status berbeda. | 1. Cari sebagian nama<br>2. Saring per chapter<br>3. Saring per status | Hasil sesuai kombinasi filter. Jumlah pada label cocok dengan baris yang tampil. | Happy path |
| CHA-01 | Chapter | P2 | CRUD chapter | Masuk sebagai admin. | 1. Buka /chapters<br>2. Tambah chapter baru<br>3. Ubah namanya<br>4. Hapus chapter yang belum punya member | Ketiga aksi berhasil dan langsung terlihat di daftar. | Happy path |
| CHA-02 | Chapter | P1 | Chapter yang masih punya member tidak bisa dihapus | Chapter dengan minimal satu member. | 1. Coba hapus chapter tersebut | Ditolak dengan pesan jelas. Member tidak boleh jadi yatim. | Negatif |
| INV-01 | Invoice | P1 | Menerbitkan invoice — alur lengkap | Admin. Member aktif ada. | 1. Buka /invoices/new<br>2. Pilih member, jenis renewal, nominal, jatuh tempo, periode<br>3. Simpan | Invoice dibuat berstatus draft dengan nomor berformat INV-&lt;tahun&gt;-NNN. Muncul di /invoices dan /invoices/{id}. | Happy path |
| INV-02 | Invoice | P1 | Nomor invoice unik pada penerbitan serentak | Admin. | 1. Terbitkan 20+ invoice bersamaan (skrip paralel / k6 skenario tulis) | Semua berhasil 201 dan setiap nomor unik. Tidak ada 500.<br>CATATAN: bug nyata — dulu 20 dari 24 penerbitan serentak gagal 500. | Balapan |
| INV-03 | Invoice | P1 | Transisi status yang diizinkan | Invoice draft. | 1. draft → sent<br>2. sent → paid<br>Ulangi jalur lain: sent → overdue → paid, dan draft → cancelled | Semua transisi di atas diterima sesuai aturan: draft→{sent,cancelled}, sent→{paid,overdue,cancelled}, overdue→{paid,cancelled}. | Happy path |
| INV-04 | Invoice | P1 | Transisi status yang dilarang ditolak | Satu invoice paid dan satu cancelled. | 1. Coba ubah invoice paid → sent<br>2. Coba ubah invoice cancelled → paid<br>3. Coba draft → paid langsung | Ketiganya ditolak dengan pesan "transisi status tidak diizinkan". Status tidak berubah. | Negatif |
| INV-05 | Invoice | P1 | Invoice lunas/batal tidak bisa diubah nominalnya | Invoice berstatus paid. | 1. PATCH nominal invoice<br>2. PATCH jatuh tempo<br>3. PATCH periode | Semua ditolak: "invoice berstatus paid tidak bisa diubah nominal/periodenya". Catatan tertutup tidak boleh ditulis ulang. | Negatif |
| INV-06 | Invoice | P1 | Dua transisi status bertabrakan | Satu invoice berstatus sent. | 1. Kirim PATCH →paid dan PATCH →cancelled bersamaan | Tepat SATU yang berhasil. Invoice tidak boleh berakhir paid sekaligus cancelled.<br>CATATAN: bug nyata — 11 dari 12 putaran, keduanya berhasil. | Balapan |
| INV-07 | Invoice | P1 | Kirim invoice ke Paper.id | Invoice draft. Kredensial Paper.id staging terpasang. | 1. Buka /invoices/{id}<br>2. Klik Kirim<br>3. Pilih kanal (email / WhatsApp)<br>4. Konfirmasi | Status menjadi sent. Tautan/nomor Paper.id tersimpan. Kanal yang DIPILIH benar-benar terkirim — bukan mengabaikan pilihan.<br>CATATAN: dulu body kosong membuat kedua kanal selalu false. | Integrasi |
| INV-08 | Invoice | P2 | Nomor invoice terbakar setelah dikirim | Invoice sudah pernah dikirim ke Paper.id. | 1. Coba kirim ulang invoice dengan nomor yang sama | Ditolak dengan pesan nomor sudah dipakai. Nomor di Paper.id bersifat permanen — pengulangan harus memakai invoice baru. | Integrasi |
| INV-09 | Invoice | P2 | Jejak audit invoice | Invoice yang sudah beberapa kali berubah status. | 1. Buka /invoices/{id}<br>2. Lihat riwayat/audit | Setiap perubahan tercatat dengan waktu dan pelaku. Urutannya benar. | Happy path |
| INV-10 | Invoice | P2 | Daftar jatuh tempo perpanjangan | Beberapa member dengan periode berakhir dalam rentang dekat. | 1. Buka /invoices/renewal-due<br>2. Ubah rentang tanggal | Hanya member yang benar-benar jatuh tempo di rentang itu yang tampil. | Happy path |
| INV-11 | Invoice | P2 | Validasi periode dan jatuh tempo | Form invoice baru. | 1. Isi periodeAkhir lebih awal dari periodeMulai<br>2. Isi nominal 0 atau negatif | Ditolak dengan pesan per-field sebelum menyentuh database. | Negatif |
| INV-12 | Invoice | P3 | Hapus invoice draft | Invoice draft yang belum pernah dikirim. | 1. Hapus invoice tersebut<br>2. Coba hapus invoice yang sudah sent/paid | Draft terhapus. Yang sudah sent/paid ditolak — riwayat keuangan tidak boleh hilang. | Negatif |
| PAY-01 | Pembayaran | P1 | Catat pembayaran manual | Invoice berstatus sent. | 1. Buka /payments atau detail invoice<br>2. Catat pembayaran penuh<br>3. Simpan | Pembayaran tercatat, invoice menjadi paid, dashboard ikut berubah. | Happy path |
| PAY-02 | Pembayaran | P1 | Pembayaran ganda pada invoice yang sama | Invoice sent. | 1. Kirim 30+ pencatatan pembayaran untuk invoice yang sama, bersamaan | Tidak ada 5xx. Invoice berakhir paid. Jumlah pembayaran tercatat masuk akal dan tidak ada nominal ganda yang tidak dijelaskan. | Balapan |
| PAY-08 | Pembayaran | P2 | Webhook Paper.id memperbarui status | Invoice sudah dikirim ke Paper.id. | 1. Kirim callback Paper.id dengan token yang benar | Status invoice terbarui sesuai payload. Tercatat di jejak audit. | Integrasi |
| DSH-01 | Dashboard & Laporan | P1 | Ringkasan dashboard cocok dengan data | Beberapa invoice dengan status beragam. | 1. Buka /dashboard<br>2. Bandingkan total & jumlah dengan /invoices dan /payments | Angka ringkasan sama persis dengan hitungan manual dari daftar. Tidak ada angka hardcoded. | Happy path |
| DSH-02 | Dashboard & Laporan | P2 | Filter rentang tanggal pada laporan | Data tersebar di beberapa bulan. | 1. Buka /reports<br>2. Pilih rentang tanggal<br>3. Ganti ke rentang tanpa data | Hasil berubah sesuai rentang. Rentang kosong menampilkan keadaan kosong yang jelas, bukan galat. | Happy path |
| DSH-03 | Dashboard & Laporan | P2 | Ekspor laporan | Ada data pada rentang terpilih. | 1. Klik ekspor<br>2. Buka berkas hasil unduhan | Berkas terunduh, terbuka tanpa rusak, isinya cocok dengan yang tampil di layar. | Happy path |
| DSH-04 | Dashboard & Laporan | P2 | Halaman mendesak (urgent) | Ada invoice lewat jatuh tempo. | 1. Buka /urgent | Menampilkan invoice yang benar-benar terlambat, terurut paling mendesak lebih dulu. | Happy path |
| SET-01 | Pengaturan | P1 | Menyimpan pengaturan aplikasi | Masuk sebagai admin. | 1. Buka /settings<br>2. Ubah salah satu nilai<br>3. Simpan<br>4. Muat ulang halaman | Nilai bertahan setelah muat ulang.<br>CATATAN: jalur PUT — dulu gagal senyap di peramban karena PUT tidak ada di Access-Control-Allow-Methods, padahal curl berhasil. | Happy path |
| SET-02 | Pengaturan | P2 | Pengaturan biaya (fee) | Admin. | 1. Ubah fee-settings<br>2. Terbitkan invoice baru | Perhitungan pada invoice baru memakai nilai fee yang baru. | Happy path |
| SET-04 | Pengaturan | P1 | Rahasia gateway tidak pernah tampil di API | Admin. | 1. GET /api/v1/app-settings<br>2. Cari kunci gateway (Xendit, Paper.id, BNI VM) | Rahasia tidak ada dalam respons — disimpan di environment, bukan app_settings. Nilai tersamar pun tetap nilai yang bisa salah dirotasi. | Keamanan |
| SET-05 | Pengaturan | P2 | Sinkronisasi | Admin. | 1. Buka /settings/sync<br>2. Jalankan sinkronisasi<br>3. Jalankan lagi segera | Berjalan sampai selesai dengan ringkasan hasil. Menjalankan dua kali tidak menggandakan data. | Integrasi |
| TOOL-01 | Alat Admin | P2 | API Console mengisi parameter otomatis | Admin, mode Backend API. | 1. Buka /api-console<br>2. Pilih beberapa endpoint berbeda | Parameter dan body terisi otomatis dengan data yang benar-benar ada. Body tampil sebagai field berlabel, bukan JSON mentah. | Happy path |
| TOOL-02 | Alat Admin | P2 | API Console mengeksekusi permintaan | API Console terbuka. | 1. Jalankan GET yang aman<br>2. Amati status, waktu, dan body respons | Respons tampil lengkap dengan kode status. Galat ditampilkan apa adanya, bukan disembunyikan. | Happy path |
| TOOL-03 | Alat Admin | P2 | Blackbox merekam lalu lintas | Admin. | 1. Buka /blackbox<br>2. Lakukan beberapa aksi di tab lain<br>3. Kembali dan segarkan | Permintaan terekam beserta body dan status. HEADER TIDAK boleh ikut terekam — di situ ada token. | Keamanan |
| TOOL-04 | Alat Admin | P3 | Bersihkan blackbox | Blackbox berisi rekaman. | 1. Klik bersihkan | Rekaman kosong. Tidak memengaruhi data bisnis. | Happy path |
| TOOL-05 | Alat Admin | P2 | Endpoint /metrics terlindungi | METRICS_TOKEN diset. | 1. GET /metrics tanpa token<br>2. GET /metrics dengan token benar | Tanpa token ditolak. Dengan token menghasilkan format Prometheus. Label TIDAK boleh memuat id member, nomor invoice, email, atau nominal. | Keamanan |
| TOOL-06 | Alat Admin | P3 | Unggah berkas | Admin. | 1. Unggah berkas dalam batas ukuran<br>2. Unggah berkas melebihi MAX_UPLOAD_SIZE<br>3. Unggah tipe berkas tak terduga | Yang sah berhasil. Yang melebihi batas ditolak dengan pesan jelas, bukan timeout. | Negatif |
| SRC-01 | Sumber Data | P1 | Beralih Data Contoh ↔ Backend API | — | 1. Di /login klik "Backend API"<br>2. Amati halaman memuat ulang<br>3. Kembali ke "Data Contoh" | Peralihan berhasil di kedua arah. Mode aktif tampak jelas. Data yang tampil sesuai sumber yang dipilih. | Happy path |
| SRC-02 | Sumber Data | P1 | Mode demo berjalan tanpa backend | Backend dimatikan. | 1. Pilih Data Contoh<br>2. Jelajahi dashboard, invoice, member, laporan | Semua halaman berfungsi penuh tanpa satu pun panggilan jaringan gagal. Ini jalur demo. | Happy path |
| SRC-03 | Sumber Data | P1 | Mode API menampilkan galat saat backend mati | Mode Backend API. Backend dimatikan. | 1. Buka /dashboard | Pesan galat yang bisa dipahami. TIDAK boleh menampilkan halaman kosong seolah datanya memang nol. | Negatif |
| RSP-01 | Responsif | P1 | Tak ada gulir horizontal di ponsel | Lebar 375px. | 1. Buka setiap rute utama satu per satu<br>2. Periksa document.scrollWidth vs clientWidth | Tidak ada rute yang meluber ke samping. Tabel lebar bergulir di dalam wadahnya sendiri, bukan mendorong halaman. | Happy path |
| RSP-02 | Responsif | P2 | Navigasi ponsel | Lebar 375px. | 1. Buka menu<br>2. Pindah halaman<br>3. Tutup menu | Menu terbuka menutup rapi, tidak terpotong, dan tidak menutupi konten setelah navigasi. | Happy path |
| RSP-03 | Responsif | P2 | Tablet dan orientasi | Lebar 768px dan 1024px. | 1. Periksa dashboard, daftar invoice, form invoice baru<br>2. Putar orientasi | Tata letak menyesuaikan tanpa elemen bertumpuk atau terpotong. | Happy path |
| RSP-04 | Responsif | P3 | Verifikasi dilakukan pada halaman yang benar-benar ter-render | — | 1. Sebelum menilai apa pun, pastikan tidak ada overlay galat Vite dan #root berisi teks | Pemeriksaan hanya sah bila halamannya benar-benar tampil.<br>CATATAN: audit responsif pernah dinyatakan "semua ok" padahal build sedang rusak — overlay galat tidak punya gulir horizontal. | Negatif |
| X-01 | Lintas Fungsi | P1 | Perjalanan penuh: member → invoice → kirim → bayar → dashboard | Sistem bersih. Admin siap. | 1. Buat chapter<br>2. Buat member<br>3. Terbitkan invoice<br>4. Kirim ke Paper.id<br>5. Buka tautan pembayaran Paper.id dari detail invoice<br>6. Lunasi (catat manual atau webhook Paper.id)<br>7. Periksa dashboard dan laporan | Setiap langkah berhasil dan status akhir konsisten di semua halaman. Ini skenario penerimaan utama. | Happy path |
| X-02 | Lintas Fungsi | P2 | Perjalanan diulang dua kali berturut-turut | Baru saja menyelesaikan X-01. | 1. Ulangi X-01 dari awal tanpa membersihkan database | Putaran kedua juga berhasil. Nomor invoice tidak bentrok.<br>CATATAN: dulu putaran kedua gagal 403 karena penomoran di-reset sementara Paper.id membakar nomor selamanya. | Negatif |
| X-03 | Lintas Fungsi | P2 | Beban campuran tetap stabil | Backend jalan, kredensial uji siap. | 1. Jalankan k6: BASE_URL=... ADMIN_EMAIL=... ADMIN_PASSWORD=... k6 run loadtest/api.js | Exit code 0. Seluruh threshold lulus, termasuk empat lantai volume. Nol nomor invoice ganda. | Beban |
| X-04 | Lintas Fungsi | P3 | Halaman tidak dikenal | — | 1. Buka /rute-yang-tidak-ada | Halaman 404 milik aplikasi dengan jalan kembali, bukan layar kosong. | Negatif |
