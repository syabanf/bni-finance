-- =============================================================================
-- BNI Finance System — skema Postgres
--
-- Skema lengkap untuk database lokal/self-hosted. Menggantikan seluruh isi
-- supabase/ (schema + migrasi 0002–0005 + rls.sql) yang sudah dilipat ke sini.
--
-- Perbedaan penting dari versi Supabase:
--   • Tidak ada RLS. Otorisasi ditegakkan backend Go lewat JWT + peran, bukan
--     database — backend memakai satu koneksi tepercaya, jadi kebijakan
--     per-baris tidak punya identitas pengguna untuk dipakai.
--   • Tabel `users` baru: menggantikan Supabase Auth.
--   • Bukti pembayaran disimpan di disk (lihat UPLOAD_DIR), bukan Storage.
--
-- Jalankan:  make db-reset      (drop + create + schema + seed)
-- =============================================================================

create extension if not exists pgcrypto;   -- gen_random_uuid()
create extension if not exists pg_trgm;    -- pencarian ILIKE '%…%'

-- --- Enum -------------------------------------------------------------------

do $$ begin
  create type invoice_type   as enum ('registration', 'renewal');
exception when duplicate_object then null; end $$;

do $$ begin
  create type invoice_status as enum ('draft', 'sent', 'paid', 'overdue', 'cancelled');
exception when duplicate_object then null; end $$;

do $$ begin
  create type member_status  as enum ('active', 'inactive', 'pending');
exception when duplicate_object then null; end $$;

do $$ begin
  create type audit_action   as enum ('created', 'sent', 'paid', 'cancelled', 'overdue', 'updated');
exception when duplicate_object then null; end $$;

do $$ begin
  create type user_role      as enum ('admin', 'user');
exception when duplicate_object then null; end $$;

-- ---------------------------------------------------------------------------
-- users — pengganti Supabase Auth
--
-- password_hash berformat  pbkdf2_sha256$<iterasi>$<salt-b64>$<hash-b64>
-- (lihat internal/auth/password.go). Tidak ada kata sandi tersimpan polos.
-- ---------------------------------------------------------------------------
create table if not exists users (
  id            uuid primary key default gen_random_uuid(),
  email         text not null unique,
  password_hash text not null,
  name          text not null,
  role          user_role not null default 'user',
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now()
);

-- Email dibandingkan tanpa membedakan huruf besar/kecil saat login.
create unique index if not exists idx_users_email_lower on users (lower(email));

-- ---------------------------------------------------------------------------
-- chapters
-- ---------------------------------------------------------------------------
create table if not exists chapters (
  id           text primary key,
  name         text not null,
  display_name text not null,
  area_name    text,
  city_name    text,
  synced_at    timestamptz not null default now()
);

create index if not exists idx_chapters_display_name on chapters (display_name);
create index if not exists idx_chapters_city_name    on chapters (city_name) where city_name is not null;

-- ---------------------------------------------------------------------------
-- members
-- ---------------------------------------------------------------------------
create table if not exists members (
  id             text primary key,
  chapter_id     text not null references chapters(id),
  name           text not null,
  email          text,
  phone          text,
  company        text,
  business_field text,
  status         member_status not null default 'active',
  joined_date    date,
  renewal_date   date,
  synced_at      timestamptz not null default now()
);

create index if not exists idx_members_chapter_name on members (chapter_id, name);
create index if not exists idx_members_status       on members (status);
create index if not exists idx_members_name_trgm    on members using gin (name gin_trgm_ops);
create index if not exists idx_members_renewal_date on members (renewal_date)
  where status = 'active' and renewal_date is not null;

-- ---------------------------------------------------------------------------
-- fee_settings — baris tunggal id = 'default'
-- ---------------------------------------------------------------------------
create table if not exists fee_settings (
  id               text primary key default 'default',
  registration_fee integer not null default 1500000,
  renewal_fee      integer not null default 1500000,
  currency         text not null default 'IDR',
  notes            text,
  updated_by       text,
  updated_at       timestamptz not null default now(),
  created_at       timestamptz not null default now()
);
insert into fee_settings (id) values ('default') on conflict do nothing;

-- ---------------------------------------------------------------------------
-- app_settings — konfigurasi key/value
--
-- Menyimpan kredensial (token BNI VM, secret Xendit), karena itu API
-- menyamarkan key yang namanya berbau rahasia — lihat domain.IsSecretKey.
-- ---------------------------------------------------------------------------
create table if not exists app_settings (
  key        text primary key,
  value      text not null,
  updated_at timestamptz not null default now()
);
insert into app_settings (key, value) values
  ('self_payment_mode',        'false'),
  ('invoice_draft_days_before','30')
on conflict (key) do nothing;

-- ---------------------------------------------------------------------------
-- invoices
-- ---------------------------------------------------------------------------
create table if not exists invoices (
  id                    uuid primary key default gen_random_uuid(),
  number                text not null unique,
  member_id             text not null references members(id),
  chapter_id            text not null references chapters(id),

  type                  invoice_type not null,
  amount                integer not null,
  currency              text not null default 'IDR',

  due_date              date not null,
  period_start          date not null,
  period_end            date not null,

  status                invoice_status not null default 'draft',

  -- Paper.id
  paper_id_invoice_id   text,
  paper_id_invoice_url  text,
  paper_id_payment_url  text,
  paper_id_sent_at      timestamptz,

  -- Xendit (Self Payment Mode)
  payment_provider      text,
  xendit_external_id    text,
  xendit_payment_id     text,
  xendit_payment_method text,
  xendit_va_bank        text,
  xendit_va_number      text,
  xendit_qris_string    text,
  xendit_payment_status text,
  xendit_expires_at     timestamptz,

  paid_at               timestamptz,
  paid_amount           integer,

  notes                 text,
  created_by            text,
  cancelled_by          text,
  cancelled_at          timestamptz,
  cancel_reason         text,

  created_at            timestamptz not null default now(),
  updated_at            timestamptz not null default now()
);

create index if not exists idx_invoices_created_at         on invoices (created_at desc);
create index if not exists idx_invoices_chapter_created    on invoices (chapter_id, created_at desc);
create index if not exists idx_invoices_status_due         on invoices (status, due_date);
create index if not exists idx_invoices_member             on invoices (member_id);
create index if not exists idx_invoices_number_trgm        on invoices using gin (number gin_trgm_ops);
create index if not exists idx_invoices_xendit_external_id on invoices (xendit_external_id)
  where xendit_external_id is not null;

-- ---------------------------------------------------------------------------
-- payments
-- ---------------------------------------------------------------------------
create table if not exists payments (
  id                  uuid primary key default gen_random_uuid(),
  invoice_id          uuid not null references invoices(id),
  amount              integer not null,
  paid_at             timestamptz not null default now(),
  payment_method      text,
  paper_id_payment_id text,
  paper_id_status     text,
  xendit_payment_id   text,
  xendit_status       text,
  proof_url           text,
  note                text,
  created_at          timestamptz not null default now()
);

create index if not exists idx_payments_invoice on payments (invoice_id);
create index if not exists idx_payments_paid_at on payments (paid_at desc);
create index if not exists idx_payments_method  on payments (payment_method);

-- ---------------------------------------------------------------------------
-- invoice_audit_log
-- ---------------------------------------------------------------------------
create table if not exists invoice_audit_log (
  id         uuid primary key default gen_random_uuid(),
  invoice_id uuid not null references invoices(id),
  action     audit_action not null,
  old_status invoice_status,
  new_status invoice_status,
  actor_id   text,
  actor_name text,
  notes      text,
  created_at timestamptz not null default now()
);

create index if not exists idx_audit_invoice_created on invoice_audit_log (invoice_id, created_at desc);
