-- ===========================================================================
-- Indeks performa untuk pola query backend (backend/internal/*/repository.go).
--
-- Temuan audit: setiap request daftar melakukan ORDER BY pada kolom yang TIDAK
-- terindeks, sehingga Postgres harus menyortir seluruh tabel setiap kali:
--   invoices  → ORDER BY created_at DESC
--   payments  → ORDER BY paid_at    DESC
-- Kolom yang sama juga dipakai filter rentang (issuedFrom/To, paidFrom/To).
--
-- Aman dijalankan berulang. Jalankan di Supabase SQL Editor.
-- ===========================================================================

-- --- invoices --------------------------------------------------------------

-- Dipakai ORDER BY pada SEMUA query daftar + filter issuedFrom/issuedTo.
create index if not exists invoices_created_at_idx on invoices (created_at desc);

-- Filter gabungan yang paling sering: per-chapter diurutkan terbaru.
create index if not exists invoices_chapter_created_idx on invoices (chapter_id, created_at desc);

-- Papan kerja penagihan: status + jatuh tempo (aging / outstanding).
create index if not exists invoices_status_due_date_idx on invoices (status, due_date);

-- --- payments --------------------------------------------------------------

-- Dipakai ORDER BY pada SEMUA query daftar + filter paidFrom/paidTo.
create index if not exists payments_paid_at_idx on payments (paid_at desc);

-- Filter metode pembayaran (rekonsiliasi per kanal).
create index if not exists payments_method_idx on payments (payment_method);

-- --- pencarian nomor invoice ------------------------------------------------
-- `number ILIKE '%x%'` punya wildcard di depan sehingga TIDAK bisa memakai
-- indeks btree. pg_trgm membuat pencarian ini terindeks.
create extension if not exists pg_trgm;
create index if not exists invoices_number_trgm_idx on invoices using gin (number gin_trgm_ops);

-- Muat ulang cache skema PostgREST
select pg_notify('pgrst', 'reload schema');
