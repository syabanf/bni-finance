-- =============================================================================
-- Data contoh untuk pengembangan lokal. Aman diulang.
-- Jangan dijalankan pada database berisi data nyata.
-- =============================================================================

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
  ('mem-001','ch-garuda',   'Budi Santoso',   'fahmi@wit.id',   '082240274833','PT Maju Bersama',   'Konstruksi',  'active',  current_date - 340, current_date + 25),
  ('mem-002','ch-garuda',   'Siti Rahayu',    'fahmi@wit.id',   '082240274833','CV Karya Abadi',    'Kuliner',     'active',  current_date - 300, current_date + 65),
  ('mem-003','ch-nusantara','Andi Wijaya',    'fahmi@wit.id',   '082240274833','PT Sinar Terang',   'Properti',    'active',  current_date - 355, current_date + 10),
  ('mem-004','ch-nusantara','Dewi Lestari',   'fahmi@wit.id',   '082240274833','Lestari Group',     'Retail',      'active',  current_date - 120, current_date + 245),
  ('mem-005','ch-merdeka',  'Rudi Hartono',   'fahmi@wit.id',   '082240274833','PT Cipta Karya',    'Manufaktur',  'active',  current_date - 200, current_date + 165),
  ('mem-006','ch-merdeka',  'Maya Puspita',   'fahmi@wit.id',   '082240274833','Puspita Consulting','Jasa',        'pending', null,               null),
  ('mem-007','ch-samudra',  'Hendra Gunawan', 'fahmi@wit.id', '082240274833','PT Bahari Jaya',    'Logistik',    'active',  current_date - 60,  current_date + 305),
  ('mem-008','ch-samudra',  'Rina Kartika',   'fahmi@wit.id',   '082240274833','Kartika Digital',   'Teknologi',   'inactive',current_date - 400, current_date - 35)
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
