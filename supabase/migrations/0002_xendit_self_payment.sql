-- ===========================================================================
-- Self Payment Mode — Xendit (Virtual Account + QRIS), native in-app
-- Apply via Supabase SQL editor.
-- ===========================================================================

-- --- invoices: kolom pembayaran Xendit ------------------------------------
alter table invoices add column if not exists payment_provider     text;          -- 'xendit' | null
alter table invoices add column if not exists xendit_external_id    text;          -- reference yg dikirim ke Xendit (pakai nomor invoice)
alter table invoices add column if not exists xendit_payment_id     text;          -- id objek VA/QR dari Xendit (dipakai webhook utk match)
alter table invoices add column if not exists xendit_payment_method text;          -- 'va' | 'qris'
alter table invoices add column if not exists xendit_va_bank        text;          -- BCA | BNI | MANDIRI | BRI
alter table invoices add column if not exists xendit_va_number      text;          -- nomor virtual account
alter table invoices add column if not exists xendit_qris_string    text;          -- payload QRIS utk render QR
alter table invoices add column if not exists xendit_payment_status text;          -- PENDING | PAID | EXPIRED
alter table invoices add column if not exists xendit_expires_at     timestamptz;   -- masa berlaku VA/QR

create index if not exists invoices_xendit_external_id_idx on invoices(xendit_external_id);
create index if not exists invoices_xendit_payment_id_idx  on invoices(xendit_payment_id);

-- --- payments: jejak transaksi Xendit -------------------------------------
alter table payments add column if not exists xendit_payment_id text;
alter table payments add column if not exists xendit_status     text;

-- --- app_settings: default Self Payment Mode (OFF) ------------------------
insert into app_settings (key, value)
values ('self_payment_mode', 'false')
on conflict (key) do nothing;

-- Reload PostgREST schema cache
select pg_notify('pgrst', 'reload schema');
