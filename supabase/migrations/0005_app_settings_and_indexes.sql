-- 0005 — Tabel app_settings + indeks untuk members & chapters
--
-- KENAPA ADA MIGRASI INI
--
-- 1. `app_settings` tidak pernah didefinisikan di mana pun. Migrasi 0002 sudah
--    melakukan `insert into app_settings (...)`, dan tiga edge function membaca
--    tabel ini (`self_payment_mode`, `invoice_draft_days_before`, token BNI VM).
--    Di database yang dibuat dari nol, 0002 akan GAGAL. Semuanya idempoten,
--    jadi aman dijalankan pada database yang tabelnya sudah dibuat manual.
--
-- 2. Migrasi 0004 mengindeks invoices & payments; members dan chapters belum —
--    padahal endpoint /members dan ringkasan dashboard memfilter kolomnya.
--
-- Jalankan di Supabase SQL Editor. Aman diulang.

-- --- app_settings ---------------------------------------------------------

create table if not exists app_settings (
  key        text primary key,
  value      text not null,
  updated_at timestamptz not null default now()
);

comment on table app_settings is
  'Konfigurasi key/value. Berisi kredensial (token BNI VM), jadi RLS harus authenticated-only — lihat rls.sql.';

-- Nilai default agar aplikasi punya perilaku yang terdefinisi sejak awal.
insert into app_settings (key, value) values
  ('self_payment_mode',        'false'),
  ('invoice_draft_days_before','30')
on conflict (key) do nothing;

-- --- members ---------------------------------------------------------------

-- Daftar member difilter per chapter dan diurutkan berdasarkan nama.
create index if not exists idx_members_chapter_name
  on members (chapter_id, name);

-- Query "jatuh tempo dalam N hari" — hanya member aktif yang relevan.
create index if not exists idx_members_renewal_date
  on members (renewal_date)
  where status = 'active' and renewal_date is not null;

create index if not exists idx_members_status
  on members (status);

-- Pencarian nama/email/company memakai ILIKE '%…%'; butuh trigram.
create extension if not exists pg_trgm;

create index if not exists idx_members_name_trgm
  on members using gin (name gin_trgm_ops);

-- --- chapters --------------------------------------------------------------

-- ORDER BY display_name pada setiap daftar chapter, dan filter kota.
create index if not exists idx_chapters_display_name
  on chapters (display_name);

create index if not exists idx_chapters_city_name
  on chapters (city_name)
  where city_name is not null;

-- --- invoice_audit_log -----------------------------------------------------

-- Timeline dibaca per invoice, terbaru dulu.
create index if not exists idx_audit_invoice_created
  on invoice_audit_log (invoice_id, created_at desc);

-- Muat ulang cache skema PostgREST
select pg_notify('pgrst', 'reload schema');
