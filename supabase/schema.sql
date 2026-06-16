-- =============================================================================
-- BNI Finance System — Supabase Schema
-- Run this in the Supabase SQL Editor (project: smahzchoqpeoxotbmmel)
-- =============================================================================

-- Enums
create type invoice_type   as enum ('registration', 'renewal');
create type invoice_status as enum ('draft', 'sent', 'paid', 'overdue', 'cancelled');
create type member_status  as enum ('active', 'inactive', 'pending');
create type audit_action   as enum ('created', 'sent', 'paid', 'cancelled', 'overdue', 'updated');

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

-- ---------------------------------------------------------------------------
-- members
-- ---------------------------------------------------------------------------
create table if not exists members (
  id          text primary key,
  chapter_id  text not null references chapters(id),
  name        text not null,
  email         text,
  phone         text,
  company       text,
  business_field text,
  status        member_status not null default 'active',
  joined_date date,
  renewal_date date,
  synced_at   timestamptz not null default now()
);
create index if not exists members_chapter_id_idx on members(chapter_id);

-- ---------------------------------------------------------------------------
-- fee_settings (single row)
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
  paper_id_invoice_id   text,
  paper_id_invoice_url  text,
  paper_id_payment_url  text,
  paper_id_sent_at      timestamptz,
  payment_provider      text,         -- 'xendit' | null
  xendit_external_id    text,
  xendit_payment_id     text,
  xendit_payment_method text,         -- 'va' | 'qris'
  xendit_va_bank        text,
  xendit_va_number      text,
  xendit_qris_string    text,
  xendit_payment_status text,         -- PENDING | PAID | EXPIRED
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
create index if not exists invoices_member_id_idx  on invoices(member_id);
create index if not exists invoices_chapter_id_idx on invoices(chapter_id);
create index if not exists invoices_status_idx     on invoices(status);
create index if not exists invoices_due_date_idx   on invoices(due_date);

-- ---------------------------------------------------------------------------
-- payments
-- ---------------------------------------------------------------------------
create table if not exists payments (
  id                  uuid primary key default gen_random_uuid(),
  invoice_id          uuid not null references invoices(id),
  amount              integer not null,
  paid_at             timestamptz not null,
  payment_method      text,
  paper_id_payment_id text,
  paper_id_status     text,
  xendit_payment_id   text,
  xendit_status       text,
  created_at          timestamptz not null default now()
);
create index if not exists payments_invoice_id_idx on payments(invoice_id);

-- ---------------------------------------------------------------------------
-- invoice_audit_log
-- ---------------------------------------------------------------------------
create table if not exists invoice_audit_log (
  id          uuid primary key default gen_random_uuid(),
  invoice_id  uuid not null references invoices(id),
  action      audit_action not null,
  old_status  invoice_status,
  new_status  invoice_status,
  actor_id    text,
  actor_name  text,
  notes       text,
  created_at  timestamptz not null default now()
);
create index if not exists audit_invoice_id_idx on invoice_audit_log(invoice_id);

-- ---------------------------------------------------------------------------
-- Row Level Security (RLS) — allow all for anon key (tighten later per role)
-- ---------------------------------------------------------------------------
alter table chapters          enable row level security;
alter table members           enable row level security;
alter table fee_settings      enable row level security;
alter table invoices          enable row level security;
alter table payments          enable row level security;
alter table invoice_audit_log enable row level security;

-- Permissive policies (anon key can read/write — tighten when auth roles are set up)
create policy "allow_all_chapters"   on chapters          for all using (true) with check (true);
create policy "allow_all_members"    on members           for all using (true) with check (true);
create policy "allow_all_fees"       on fee_settings      for all using (true) with check (true);
create policy "allow_all_invoices"   on invoices          for all using (true) with check (true);
create policy "allow_all_payments"   on payments          for all using (true) with check (true);
create policy "allow_all_audit"      on invoice_audit_log for all using (true) with check (true);

-- ---------------------------------------------------------------------------
-- Helper function: auto-update invoices.updated_at
-- ---------------------------------------------------------------------------
create or replace function set_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger invoices_updated_at
  before update on invoices
  for each row execute function set_updated_at();

create trigger fee_settings_updated_at
  before update on fee_settings
  for each row execute function set_updated_at();
