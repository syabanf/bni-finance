-- =============================================================================
-- Seed 2026-08-24 — chapter baru beserta membernya
--
-- Empat chapter di kota yang belum terwakili, masing-masing dengan lima member.
-- Sebelum ini seluruh data contoh hanya menyentuh enam kota di Jawa, Bali, dan
-- Kalimantan; laporan per chapter dan filter wilayah tidak pernah teruji atas
-- sebaran yang benar-benar lebar.
--
-- SEMUA MEMBER DI SINI TIDAK PUNYA INVOICE, sama seperti seed member sebelumnya.
-- Halaman "Buat Invoice" hanya menampilkan member yang belum punya invoice
-- pendaftaran aktif — kalau setiap member contoh sudah ditagih, tidak ada lagi
-- yang bisa dipilih untuk mencoba alur pengiriman.
--
-- Berkas ini BERDIRI SENDIRI: ia membuat chapternya sendiri, jadi tidak
-- bergantung pada init.sql maupun seed lain yang kebetulan dijalankan lebih
-- dulu. Idempoten — aman dijalankan berulang kali, dalam urutan apa pun.
--
--   make db-seed-baru
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seeds/2026-08-24-chapter-baru-beserta-member.sql
-- =============================================================================

begin;

-- --- Chapter baru ------------------------------------------------------------
insert into chapters (id, name, display_name, area_name, city_name) values
  ('ch-bhinneka',     'Bhinneka',     'BNI Bhinneka',     'Medan Kota',      'Medan'),
  ('ch-kartika',      'Kartika',      'BNI Kartika',      'Semarang Tengah', 'Semarang'),
  ('ch-rinjani',      'Rinjani',      'BNI Rinjani',      'Mataram Barat',   'Mataram'),
  ('ch-khatulistiwa', 'Khatulistiwa', 'BNI Khatulistiwa', 'Pontianak Kota',  'Pontianak')
on conflict (id) do nothing;

-- --- Member ------------------------------------------------------------------
-- KONTAK SENGAJA SAMA untuk semuanya, mengikuti aturan yang sudah berlaku di
-- db/init.sql: menerbitkan invoice dengan kanal WhatsApp/email menyala membuat
-- Paper.id benar-benar mengirim ke nomor dan alamat yang tertulis di sini. Data
-- karangan yang berbeda-beda berarti pesan uji coba mendarat di ponsel orang
-- lain yang kebetulan memilikinya.
--
-- Ganti bila kontak ujinya berganti — jangan dikembalikan menjadi acak.
--
-- Tiap chapter sengaja berisi campuran, supaya satu chapter saja sudah cukup
-- untuk menguji kedua jenis invoice sekaligus:
--
--   2x pending, joined_date null  -> calon member, butuh invoice PENDAFTARAN
--   2x active, renewal dekat      -> butuh invoice RENEWAL
--   1x active, renewal masih jauh -> kontrol; tidak boleh muncul sebagai jatuh tempo
insert into members (id, chapter_id, name, email, phone, company, business_field, status, joined_date, renewal_date) values
  -- BNI Bhinneka — Medan
  ('mem-017','ch-bhinneka','Tengku Iskandar','muhfahmifm@gmail.com','082240274833','Iskandar Prima',      'Perkebunan',   'pending', null,                    null),
  ('mem-018','ch-bhinneka','Melati Sinaga',  'fahmi@wit.id',        '082240274833','Sinaga Bersaudara',   'Kuliner',      'pending', null,                    null),
  ('mem-019','ch-bhinneka','Ronald Sitorus', 'muhfahmifm@gmail.com','082240274833','PT Deli Makmur',      'Distribusi',   'active',  date '2026-08-24' - 356, date '2026-08-24' + 9),
  ('mem-020','ch-bhinneka','Farida Lubis',   'fahmi@wit.id',        '082240274833','Lubis Properti',      'Properti',     'active',  date '2026-08-24' - 349, date '2026-08-24' + 16),
  ('mem-021','ch-bhinneka','Anwar Nasution', 'muhfahmifm@gmail.com','082240274833','CV Sumut Teknik',     'Manufaktur',   'active',  date '2026-08-24' - 45,  date '2026-08-24' + 320),

  -- BNI Kartika — Semarang
  ('mem-022','ch-kartika','Sri Handayani',   'fahmi@wit.id',        '082240274833','Handayani Batik',     'Tekstil',      'pending', null,                    null),
  ('mem-023','ch-kartika','Bagas Susilo',    'muhfahmifm@gmail.com','082240274833','PT Jateng Logistik',  'Logistik',     'pending', null,                    null),
  ('mem-024','ch-kartika','Ratna Wijayanti', 'fahmi@wit.id',        '082240274833','Wijayanti Farma',     'Kesehatan',    'active',  date '2026-08-24' - 353, date '2026-08-24' + 12),
  ('mem-025','ch-kartika','Joko Prasetyo',   'muhfahmifm@gmail.com','082240274833','Prasetyo Konstruksi', 'Konstruksi',   'active',  date '2026-08-24' - 346, date '2026-08-24' + 19),
  ('mem-026','ch-kartika','Endah Puspitasari','fahmi@wit.id',       '082240274833','Puspita Digital',     'Teknologi',    'active',  date '2026-08-24' - 20,  date '2026-08-24' + 345),

  -- BNI Rinjani — Mataram
  ('mem-027','ch-rinjani','Lalu Ahmad',      'muhfahmifm@gmail.com','082240274833','Ahmad Mutiara',       'Perhiasan',    'pending', null,                    null),
  ('mem-028','ch-rinjani','Baiq Salma',      'fahmi@wit.id',        '082240274833','Salma Tour',          'Pariwisata',   'pending', null,                    null),
  ('mem-029','ch-rinjani','Wayan Sudirja',   'muhfahmifm@gmail.com','082240274833','PT Lombok Bahari',    'Perikanan',    'active',  date '2026-08-24' - 351, date '2026-08-24' + 14),
  ('mem-030','ch-rinjani','Dinda Maharani',  'fahmi@wit.id',        '082240274833','Maharani Kriya',      'Kerajinan',    'active',  date '2026-08-24' - 344, date '2026-08-24' + 21),
  ('mem-031','ch-rinjani','Haris Munandar',  'muhfahmifm@gmail.com','082240274833','Munandar Agro',       'Agrikultur',   'active',  date '2026-08-24' - 70,  date '2026-08-24' + 295),

  -- BNI Khatulistiwa — Pontianak
  ('mem-032','ch-khatulistiwa','Rudi Tanjaya',   'fahmi@wit.id',        '082240274833','Tanjaya Kayu',    'Kehutanan',    'pending', null,                    null),
  ('mem-033','ch-khatulistiwa','Siska Halim',    'muhfahmifm@gmail.com','082240274833','Halim Elektronik','Retail',       'pending', null,                    null),
  ('mem-034','ch-khatulistiwa','Bambang Setiadi','fahmi@wit.id',        '082240274833','PT Kapuas Energi','Energi',       'active',  date '2026-08-24' - 355, date '2026-08-24' + 10),
  ('mem-035','ch-khatulistiwa','Novi Andriani',  'muhfahmifm@gmail.com','082240274833','Andriani Katering','Kuliner',     'active',  date '2026-08-24' - 348, date '2026-08-24' + 17),
  ('mem-036','ch-khatulistiwa','Yusuf Maulana',  'fahmi@wit.id',        '082240274833','Maulana Percetakan','Percetakan', 'active',  date '2026-08-24' - 15,  date '2026-08-24' + 350)
on conflict (id) do nothing;

commit;

-- --- Pemeriksaan mandiri -----------------------------------------------------
-- Dua hal yang membuat seed ini gagal diam-diam kalau tidak diperiksa: member
-- yang ternyata sudah punya invoice (tidak akan muncul di daftar "Buat Invoice"),
-- dan chapter yang berakhir tanpa satu pun member (laporan per chapter kosong
-- tanpa alasan yang terlihat).
do $$
declare
  ber_invoice int;
  chapter_kosong text;
begin
  select count(*) into ber_invoice
  from invoices
  where member_id between 'mem-017' and 'mem-036';

  if ber_invoice > 0 then
    raise warning 'ADA % invoice menempel pada member seed ini — member itu tidak akan muncul di daftar "Buat Invoice"', ber_invoice;
  end if;

  select string_agg(c.id, ', ') into chapter_kosong
  from chapters c
  where c.id in ('ch-bhinneka','ch-kartika','ch-rinjani','ch-khatulistiwa')
    and not exists (select 1 from members m where m.chapter_id = c.id);

  if chapter_kosong is not null then
    raise warning 'chapter tanpa member: %', chapter_kosong;
  else
    raise notice 'seed 2026-08-24: 4 chapter baru, 20 member, semuanya tanpa invoice';
  end if;
end $$;
