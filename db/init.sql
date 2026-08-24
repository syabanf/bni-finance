-- =============================================================================
-- db/init.sql — SATU berkas untuk menyiapkan database dari nol.
--
--     cd backend && make db-init
--
-- Menggantikan pasangan schema.sql + seed.sql yang dulu harus dijalankan
-- berurutan. Satu perintah, satu berkas, satu urutan yang benar.
--
-- Seluruhnya IDEMPOTEN dan aman diulang:
--
--   1. SKEMA        create ... if not exists — selalu dijalankan.
--   2. AKUN AWAL    hanya bila tabel users MASIH KOSONG.
--   3. DATA CONTOH  hanya bila tabel members MASIH KOSONG.
--
-- Penjaga "hanya bila kosong" itu yang membuat berkas ini tidak berbahaya bila
-- tidak sengaja dijalankan pada database berisi data nyata: skemanya menyesuaikan,
-- tetapi tidak satu baris pun akun atau data contoh yang ditambahkan.
--
-- Tidak memakai meta-command psql (\if, \echo) supaya bisa diterapkan lewat psql
-- MAUPUN klien lain seperti pgx.
--
-- -----------------------------------------------------------------------------
-- PERINGATAN — KREDENSIAL PENGEMBANGAN
--
-- Bagian 2 membuat akun dengan kata sandi yang TERTULIS DI BERKAS INI dan sama
-- di setiap salinan repo. Itu disengaja untuk pengembangan dan demo, dan itu
-- pula sebabnya berkas ini TIDAK BOLEH dijalankan pada database produksi yang
-- masih kosong.
--
-- Untuk produksi, jangan pakai bagian 2. Backend membuat admin awalnya sendiri
-- dari SEED_ADMIN_EMAIL/SEED_ADMIN_PASSWORD di environment (EnsureSeedAdmin),
-- yang juga hanya berjalan saat tabel users kosong.
-- =============================================================================


-- =============================================================================
-- 1. SKEMA
-- =============================================================================

-- =============================================================================
-- BNI Finance System — skema Postgres
--
-- Skema lengkap untuk database lokal/self-hosted. Satu berkas, idempoten:
-- skema selalu diterapkan, akun awal dan data contoh hanya bila tabelnya kosong.
--
-- Dua hal yang perlu diketahui sebelum membacanya:
--   • TIDAK ADA row-level security, dan itu bukan kelalaian. Backend menyambung
--     sebagai SATU peran tepercaya, jadi database melihat satu identitas saja
--     dan tidak punya apa pun untuk dipakai sebagai kunci kebijakan per-baris.
--     Seluruh otorisasi ditegakkan backend Go lewat JWT + peran.
--   • Bukti pembayaran disimpan di disk (lihat UPLOAD_DIR), bukan di database.
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
-- users — akun aplikasi
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
-- Bawaan aman. `do nothing` menjaga instalasi yang sudah jalan: nilai yang
-- sudah diubah operator tidak ditimpa.
--
-- Kedua kunci paperid_send_* mengatur apakah Paper.id benar-benar mengantar
-- invoice ke member. Bawaannya NYALA: invoice yang tidak pernah sampai ke
-- member bukan hasil yang lebih ringan, melainkan kegagalan diam-diam yang
-- tetap dilaporkan sukses. Hanya nilai 'false' yang mematikannya, sehingga
-- mematikan kanal selalu keputusan seseorang.
insert into app_settings (key, value) values
  ('self_payment_mode',        'false'),
  ('invoice_draft_days_before','30'),
  ('invoice_due_days_after',   '30'),
  ('paperid_send_email',       'true'),
  ('paperid_send_whatsapp',    'true')
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


-- Rekaman panggilan integrasi pihak ketiga (Paper.id, BNI VM) — kotak hitam.
--
-- Dulu hanya ring buffer di memori dan hilang saat restart. Itu cukup untuk
-- "apa yang barusan terjadi", tetapi tidak untuk pertanyaan yang sebenarnya
-- diajukan orang: "invoice ini dikirim kapan, dan waktu itu Paper.id menjawab
-- apa" — yang hampir selalu ditanyakan berhari-hari kemudian, setelah proses
-- sudah lama di-restart.
--
-- HANYA BODY, TIDAK PERNAH HEADER. Kredensial Paper.id (client_id/client_secret)
-- dan bearer token hidup di header, jadi secara konstruksi tidak bisa sampai ke
-- tabel ini. Perhatikan tetap: body permintaan Paper.id memuat nama, email, dan
-- telepon member, sehingga tabel ini menyimpan data pribadi dan hanya boleh
-- dibaca admin — sama seperti halaman yang menampilkannya.
-- Penghitung pengingat per invoice.
--
-- Paper.id membakar nomor invoice secara permanen: mengirim ulang dengan nomor
-- yang sama ditolak "nomor sudah dipakai". Pengingat karena itu memakai nomor
-- turunan — INV-2026-001-R1, -R2, dan seterusnya — sementara nomor kanonik di
-- sistem kita tidak berubah. Penghitung ini yang menentukan sufiksnya.
-- Pengingat adalah kejadian tersendiri di jejak audit, bukan penerbitan ulang.
-- Memakai 'sent' akan berbohong: riwayatnya terbaca seolah invoice diterbitkan
-- dua kali, padahal tagihannya satu dan hanya diingatkan.
do $$
begin
  if not exists (
    select 1 from pg_enum e join pg_type t on t.oid = e.enumtypid
    where t.typname = 'audit_action' and e.enumlabel = 'reminded'
  ) then
    alter type audit_action add value 'reminded';
  end if;
end $$;

alter table invoices
  add column if not exists paper_id_reminder_count integer not null default 0;


-- ---------------------------------------------------------------------------
-- Peran berlingkup chapter — ST dan MC
--
-- ST (Secretary/Treasurer) dan MC (Membership Committee) adalah pengurus SATU
-- chapter, bukan nasional. Keduanya login, dan keduanya hanya boleh melihat
-- chapternya sendiri.
--
-- Lingkupnya disimpan di users.chapter_id, dan null berarti nasional — itulah
-- yang dipakai admin dan user. Kolomnya nullable dengan sengaja: memaksanya
-- not null akan menuntut chapter untuk admin, yang tidak punya satu pun.
--
-- Perlu diketahui: TIDAK ADA row-level security yang menegakkan lingkup ini.
-- Backend menyambung sebagai satu peran tepercaya, jadi database melihat satu
-- identitas saja. Seluruh pembatasan ada di kode Go, dan tes batas akseslah
-- satu-satunya bukti bahwa ia benar-benar ada.
-- ---------------------------------------------------------------------------
do $$
begin
  if not exists (
    select 1 from pg_enum e join pg_type t on t.oid = e.enumtypid
    where t.typname = 'user_role' and e.enumlabel = 'st'
  ) then
    alter type user_role add value 'st';
  end if;
  if not exists (
    select 1 from pg_enum e join pg_type t on t.oid = e.enumtypid
    where t.typname = 'user_role' and e.enumlabel = 'mc'
  ) then
    alter type user_role add value 'mc';
  end if;
end $$;

alter table users
  add column if not exists chapter_id text references chapters(id);

create index if not exists idx_users_chapter on users (chapter_id);


-- ---------------------------------------------------------------------------
-- Kolom uang dilebarkan ke bigint
--
-- Go menyimpan seluruh nominal sebagai int64, Postgres menerimanya sebagai
-- integer (int4, maksimum 2.147.483.647). Selisih itu tidak pernah terlihat
-- sampai ada yang memasukkan angka lebih besar — lalu API menjawab 500
-- "terjadi kesalahan pada server":
--
--   unable to encode 9223372036854775807 into binary format for int4 (OID 23)
--
-- Salah ketik menambah satu nol sudah cukup, dan pesannya menyuruh orang
-- mencari kerusakan server yang tidak pernah ada. Kedua sisi disamakan di sini,
-- ke tipe yang memang dipakai kodenya.
-- ---------------------------------------------------------------------------
alter table invoices      alter column amount           type bigint;
alter table invoices      alter column paid_amount      type bigint;
alter table payments      alter column amount           type bigint;
alter table fee_settings  alter column registration_fee type bigint;
alter table fee_settings  alter column renewal_fee      type bigint;


create table if not exists integration_calls (
  id          bigserial   primary key,
  occurred_at timestamptz not null default now(),
  integration text        not null,
  direction   text        not null,
  method      text        not null,
  url         text        not null,
  request     jsonb,
  response    jsonb,
  status      integer     not null default 0,
  success     boolean     not null,
  duration_ms bigint      not null default 0,
  error       text
);

-- Halaman blackbox selalu membaca terbaru-dulu, dan pemangkasan retensi
-- memakai urutan yang sama.
create index if not exists idx_integration_calls_recent
  on integration_calls (occurred_at desc, id desc);

-- Penyaringan per integrasi dan "gagal saja" adalah dua filter yang dipakai
-- saat menelusuri masalah.
create index if not exists idx_integration_calls_integration
  on integration_calls (integration, occurred_at desc);
create index if not exists idx_integration_calls_failed
  on integration_calls (occurred_at desc) where not success;


-- =============================================================================
-- 2. AKUN AWAL — hanya bila tabel users masih kosong
--
-- Emailnya sengaja sama persis dengan akun demo pada mode Data Contoh
-- (src/services/mock/authRepository.ts), sehingga berpindah antara Data Contoh
-- dan Backend API tidak menuntut kredensial berbeda — dan skenario QA AUTH-01
-- berjalan di kedua mode.
--
--     admin@bni-finance.com   admin123   peran admin
--     user@bni-finance.com    user123    peran user
--
-- Hash dibangkitkan oleh auth.HashPassword: pbkdf2_sha256, 600.000 iterasi.
-- =============================================================================

insert into users (email, password_hash, name, role)
select * from (values
  ('admin@bni-finance.com', 'pbkdf2_sha256$600000$pcdAmM7g/k6c76RFc0wInw$zb0Xz7WyApNl4/U8yrSUwrIVf1bsC0AZljajOhe2Z+o', 'Admin Nasional', 'admin'::user_role),
  ('user@bni-finance.com',  'pbkdf2_sha256$600000$vdiDsSUR+cMP1RrISviV5Q$7j279skDDr8+vTXfB2dXOf16bNU+9JBfYhcoYG1lDh4', 'User BNI',       'user'::user_role)
) as awal(email, password_hash, name, role)
where not exists (select 1 from users);


-- =============================================================================
-- 3. DATA CONTOH — hanya bila tabel members masih kosong
-- =============================================================================

do $$
begin
  if exists (select 1 from members) then
    raise notice 'members sudah berisi data — bagian data contoh dilewati';
    return;
  end if;
  
  insert into chapters (id, name, display_name, area_name, city_name) values
    ('ch-garuda',   'Garuda',   'BNI Garuda',   'Jakarta Pusat',  'Jakarta'),
    ('ch-nusantara','Nusantara','BNI Nusantara','Jakarta Selatan','Jakarta'),
    ('ch-merdeka',  'Merdeka',  'BNI Merdeka',  'Bandung Kota',   'Bandung'),
    ('ch-samudra',  'Samudra',  'BNI Samudra',  'Surabaya Timur', 'Surabaya')
  on conflict (id) do nothing;
  
  -- Semua member memakai NOMOR DAN EMAIL YANG SAMA, dan itu disengaja.
  --
  -- Menerbitkan invoice dengan kanal WhatsApp/email menyala membuat Paper.id
  -- benar-benar mengirim ke nomor dan alamat yang tertulis di sini. Data karangan
  -- yang berbeda-beda berarti pesan uji coba mendarat di ponsel atau kotak masuk
  -- orang lain yang kebetulan memilikinya. Dengan satu kontak milik tim, uji coba
  -- hanya sampai ke kita sendiri.
  --
  -- Ganti bila kontak ujinya berganti — jangan dikembalikan menjadi acak.
  insert into members (id, chapter_id, name, email, phone, company, business_field, status, joined_date, renewal_date) values
    ('mem-001','ch-garuda',   'Budi Santoso',   'muhfahmifm@gmail.com', '082240274833','PT Maju Bersama',   'Konstruksi',  'active',  current_date - 340, current_date + 25),
    ('mem-002','ch-garuda',   'Siti Rahayu',    'fahmi@wit.id',         '082240274833','CV Karya Abadi',    'Kuliner',     'active',  current_date - 300, current_date + 65),
    ('mem-003','ch-nusantara','Andi Wijaya',    'muhfahmifm@gmail.com', '082240274833','PT Sinar Terang',   'Properti',    'active',  current_date - 355, current_date + 10),
    ('mem-004','ch-nusantara','Dewi Lestari',   'fahmi@wit.id',         '082240274833','Lestari Group',     'Retail',      'active',  current_date - 120, current_date + 245),
    ('mem-005','ch-merdeka',  'Rudi Hartono',   'muhfahmifm@gmail.com', '082240274833','PT Cipta Karya',    'Manufaktur',  'active',  current_date - 200, current_date + 165),
    ('mem-006','ch-merdeka',  'Maya Puspita',   'fahmi@wit.id',         '082240274833','Puspita Consulting','Jasa',        'pending', null,               null),
    ('mem-007','ch-samudra',  'Hendra Gunawan', 'muhfahmifm@gmail.com', '082240274833','PT Bahari Jaya',    'Logistik',    'active',  current_date - 60,  current_date + 305),
    ('mem-008','ch-samudra',  'Rina Kartika',   'fahmi@wit.id',         '082240274833','Kartika Digital',   'Teknologi',   'inactive',current_date - 400, current_date - 35)
  on conflict (id) do nothing;
  
  -- Invoice contoh yang mencakup setiap status, supaya dashboard dan filter
  -- punya sesuatu untuk ditampilkan sejak awal.
  insert into invoices (number, member_id, chapter_id, type, amount, due_date, period_start, period_end, status, paid_at, paid_amount, created_at)
  select * from (values
    ('INV-2026-001','mem-001','ch-garuda',   'renewal'::invoice_type,      1500000, current_date + 25, current_date + 25, current_date + 390, 'sent'::invoice_status,      null::timestamptz, null::integer, now() - interval '5 days'),
    ('INV-2026-002','mem-002','ch-garuda',   'renewal'::invoice_type,      1500000, current_date - 10, current_date + 65, current_date + 430, 'overdue'::invoice_status,   null,              null,          now() - interval '40 days'),
    ('INV-2026-003','mem-003','ch-nusantara','renewal'::invoice_type,      1500000, current_date + 10, current_date + 10, current_date + 375, 'paid'::invoice_status,      now() - interval '2 days', 1500000, now() - interval '12 days'),
    ('INV-2026-004','mem-004','ch-nusantara','registration'::invoice_type, 2000000, current_date + 30, current_date,      current_date + 365, 'draft'::invoice_status,     null,              null,          now() - interval '1 day'),
    ('INV-2026-005','mem-005','ch-merdeka',  'renewal'::invoice_type,      1500000, current_date + 45, current_date + 45, current_date + 410, 'paid'::invoice_status,      now() - interval '20 days',1500000, now() - interval '45 days'),
    ('INV-2026-006','mem-007','ch-samudra',  'registration'::invoice_type, 2000000, current_date + 20, current_date,      current_date + 365, 'cancelled'::invoice_status, null,              null,          now() - interval '8 days')
  ) as v(number, member_id, chapter_id, type, amount, due_date, period_start, period_end, status, paid_at, paid_amount, created_at)
  where not exists (select 1 from invoices where invoices.number = v.number);
  
  -- Pembayaran untuk invoice yang sudah lunas.
  insert into payments (invoice_id, amount, paid_at, payment_method)
  select i.id, i.amount, i.paid_at, 'bank_transfer'
  from invoices i
  where i.status = 'paid'
    and not exists (select 1 from payments p where p.invoice_id = i.id);
  
  -- Jejak audit awal, supaya timeline tidak kosong.
  insert into invoice_audit_log (invoice_id, action, new_status, actor_name, created_at)
  select i.id, 'created'::audit_action, 'draft'::invoice_status, 'Seed', i.created_at
  from invoices i
  where not exists (select 1 from invoice_audit_log a where a.invoice_id = i.id);
end
$$;
