-- =============================================================================
-- Seed 2026-09-11 — visitor (tamu yang belum mendaftar)
--
-- Status `visitor` menandai orang yang datang ke pertemuan chapter tapi BELUM
-- menjadi anggota. Tanpa satu pun baris berstatus itu, filternya kosong dan
-- tidak ada yang bisa memastikan tamu benar-benar terpisah dari member —
-- termasuk memastikan ia tidak ikut tertagih.
--
-- ENUM DITAMBAHKAN DI LUAR TRANSAKSI, dan itu bukan gaya penulisan.
-- `alter type … add value` boleh berjalan di dalam blok transaksi, tapi nilai
-- barunya TIDAK bisa dipakai sampai transaksi itu commit. Menaruhnya setelah
-- `begin;` membuat INSERT di bawahnya gagal dengan
--   ERROR: unsafe use of new value "visitor" of enum type member_status
-- pada basis data yang enum-nya belum punya nilai itu — yaitu persis basis data
-- yang paling butuh berkas ini.
--
-- Idempoten: aman dijalankan berulang kali.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seeds/2026-09-11-visitor.sql
-- =============================================================================

do $$
begin
  if not exists (
    select 1 from pg_enum e join pg_type t on t.oid = e.enumtypid
    where t.typname = 'member_status' and e.enumlabel = 'visitor'
  ) then
    alter type member_status add value 'visitor';
  end if;
end $$;

begin;

-- Tamu tersebar di beberapa chapter supaya filter per chapter punya lebih dari
-- satu bentuk untuk diuji.
--
-- renewal_date SENGAJA null. Tamu belum punya keanggotaan, jadi tidak ada yang
-- jatuh tempo — dan kolom itulah yang dibaca pemanen renewal. Mengisinya
-- dengan tanggal apa pun berarti menaruh tamu di antrean tagihan lewat pintu
-- yang tidak dijaga penyaring status.
insert into members (id, chapter_id, name, email, phone, company, business_field, status, joined_date, renewal_date) values
  ('mem-037','ch-garuda',   'Bayu Anggara',   'muhfahmifm@gmail.com','082240274833','Anggara Digital',  'Teknologi',  'visitor', date '2026-09-11' - 14, null),
  ('mem-038','ch-garuda',   'Nadia Prameswari','fahmi@wit.id',       '082240274833','Prameswari Studio','Kreatif',    'visitor', date '2026-09-11' - 7,  null),
  ('mem-039','ch-nusantara','Reza Firmansyah', 'muhfahmifm@gmail.com','082240274833','Firman Teknik',   'Manufaktur', 'visitor', date '2026-09-11' - 21, null),
  ('mem-040','ch-bhinneka', 'Clara Simanjuntak','fahmi@wit.id',      '082240274833','Simanjuntak Tour', 'Pariwisata', 'visitor', date '2026-09-11' - 3,  null)
on conflict (id) do nothing;

commit;
