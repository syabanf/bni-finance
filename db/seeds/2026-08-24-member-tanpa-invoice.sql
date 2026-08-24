-- =============================================================================
-- Seed 2026-08-24 — member yang BELUM punya invoice
--
-- Terpisah dari db/init.sql dengan sengaja. init.sql hanya mengisi data contoh
-- saat tabel `members` masih kosong; pada basis data yang sudah berisi, ia tidak
-- menambahkan apa pun. Berkas ini justru untuk keadaan itu — menambah member ke
-- basis data yang sudah hidup, tanpa menyentuh satu baris pun yang sudah ada.
--
-- SEMUA MEMBER DI SINI TIDAK PUNYA INVOICE, dan itu intinya. Halaman "Buat
-- Invoice" hanya menampilkan "member yang belum punya invoice pendaftaran
-- aktif"; setelah seluruh member contoh lama ditagih, daftarnya habis dan tidak
-- ada lagi yang bisa dipilih untuk mencoba alur pengiriman.
--
-- Idempoten: aman dijalankan berulang kali.
--
--   make db-seed-baru
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seeds/2026-08-24-member-tanpa-invoice.sql
-- =============================================================================

begin;

-- --- Chapter -----------------------------------------------------------------
-- Dua chapter baru di kota yang belum terwakili, supaya laporan per chapter dan
-- filter wilayah punya lebih dari satu bentuk untuk diuji.
--
-- Keempat chapter LAMA ikut disebut di sini, dan itu bukan duplikasi yang
-- kelupaan. Versi pertama berkas ini hanya membuat dua yang baru lalu menyisipkan
-- member ke ch-garuda — dan langsung gagal:
--
--   violates foreign key constraint "members_chapter_id_fkey"
--   Key (chapter_id)=(ch-garuda) is not present in table "chapters"
--
-- karena basis datanya baru saja di-truncate oleh integration test. Seed yang
-- hanya jalan kalau init.sql kebetulan sudah dijalankan lebih dulu adalah seed
-- yang gagal tepat saat paling dibutuhkan. Dengan on conflict do nothing,
-- menyebutkan semuanya tidak menimpa apa pun dan membuat berkas ini berdiri
-- sendiri.
insert into chapters (id, name, display_name, area_name, city_name) values
  ('ch-garuda',   'Garuda',   'BNI Garuda',    'Jakarta Pusat',   'Jakarta'),
  ('ch-nusantara','Nusantara','BNI Nusantara', 'Jakarta Selatan', 'Jakarta'),
  ('ch-merdeka',  'Merdeka',  'BNI Merdeka',   'Bandung Kota',    'Bandung'),
  ('ch-samudra',  'Samudra',  'BNI Samudra',   'Surabaya Timur',  'Surabaya'),
  ('ch-cakrawala','Cakrawala','BNI Cakrawala', 'Denpasar Selatan','Denpasar'),
  ('ch-mahakam',  'Mahakam',  'BNI Mahakam',   'Samarinda Kota',  'Samarinda')
on conflict (id) do nothing;

-- --- Member tanpa invoice ----------------------------------------------------
-- KONTAK SENGAJA SAMA untuk semuanya, mengikuti aturan yang sudah berlaku di
-- db/init.sql: menerbitkan invoice dengan kanal WhatsApp/email menyala membuat
-- Paper.id benar-benar mengirim ke nomor dan alamat yang tertulis di sini. Data
-- karangan yang berbeda-beda berarti pesan uji coba mendarat di ponsel orang
-- lain yang kebetulan memilikinya.
--
-- Ganti bila kontak ujinya berganti — jangan dikembalikan menjadi acak.
--
-- Tiga bentuk, supaya kedua jenis invoice bisa diuji:
--
--   pending, joined_date null   -> calon member, butuh invoice PENDAFTARAN
--   active, renewal sudah dekat -> butuh invoice RENEWAL
--   active, renewal masih jauh  -> kontrol; tidak boleh muncul di daftar jatuh tempo
insert into members (id, chapter_id, name, email, phone, company, business_field, status, joined_date, renewal_date) values
  -- calon member: belum bergabung, menunggu invoice pendaftaran
  ('mem-009','ch-cakrawala','Gita Anindya',    'muhfahmifm@gmail.com','082240274833','Anindya Kreatif',    'Periklanan',   'pending', null,                    null),
  ('mem-010','ch-cakrawala','Bayu Prakoso',    'fahmi@wit.id',        '082240274833','PT Samudera Biru',   'Pariwisata',   'pending', null,                    null),
  ('mem-011','ch-mahakam',  'Nadia Rahmawati', 'muhfahmifm@gmail.com','082240274833','Rahma Nusantara',    'Agrikultur',   'pending', null,                    null),
  ('mem-012','ch-mahakam',  'Fajar Nugroho',   'fahmi@wit.id',        '082240274833','CV Borneo Mandiri',  'Pertambangan', 'pending', null,                    null),

  -- sudah bergabung, renewal dekat — butuh invoice perpanjangan
  ('mem-013','ch-garuda',   'Laras Wulandari', 'muhfahmifm@gmail.com','082240274833','Wulandari Tekstil',  'Tekstil',      'active',  date '2026-08-24' - 358, date '2026-08-24' + 7),
  ('mem-014','ch-nusantara','Reza Firmansyah', 'fahmi@wit.id',        '082240274833','PT Cahaya Digital',  'Teknologi',    'active',  date '2026-08-24' - 351, date '2026-08-24' + 14),
  ('mem-015','ch-merdeka',  'Ayu Kusuma',      'muhfahmifm@gmail.com','082240274833','Kusuma Farmasi',     'Kesehatan',    'active',  date '2026-08-24' - 344, date '2026-08-24' + 21),

  -- kontrol: renewal masih jauh, tidak boleh muncul sebagai jatuh tempo
  ('mem-016','ch-samudra',  'Yoga Pratama',    'fahmi@wit.id',        '082240274833','Pratama Logistik',   'Logistik',     'active',  date '2026-08-24' - 30,  date '2026-08-24' + 335)
on conflict (id) do nothing;

commit;

-- --- Pemeriksaan mandiri -----------------------------------------------------
-- Kalau salah satu dari member ini ternyata PUNYA invoice, seluruh maksud berkas
-- ini gugur — dan lebih baik ketahuan saat dijalankan daripada saat daftar
-- "Buat Invoice" ternyata tetap kosong.
do $$
declare n int;
begin
  select count(*) into n
  from invoices
  where member_id between 'mem-009' and 'mem-016';

  if n > 0 then
    raise warning 'ADA % invoice menempel pada member seed 2026-08-24 — member itu tidak akan muncul di daftar "Buat Invoice"', n;
  else
    raise notice 'seed 2026-08-24: 8 member tanpa invoice, 2 chapter baru';
  end if;
end $$;
