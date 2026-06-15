-- =============================================================================
-- BNI Finance — seed master data (chapters + members)
-- Run AFTER schema.sql, in the Supabase SQL Editor. Safe to re-run.
-- =============================================================================

insert into chapters (id,name,display_name,area_name,city_name) values
  ('ch-garuda','garuda','Garuda','Jakarta Pusat','Jakarta'),
  ('ch-magnify','magnify','Magnify','Jakarta Selatan','Jakarta'),
  ('ch-amplify','amplify','Amplify','Bandung Kota','Bandung'),
  ('ch-rise','rise','Rise','Surabaya Timur','Surabaya'),
  ('ch-glorify','glorify','Glorify','Semarang','Semarang'),
  ('ch-victory','victory','Victory','Denpasar','Bali')
on conflict (id) do nothing;

insert into members (id,chapter_id,name,email,phone,status,joined_date) values
  ('m001','ch-garuda','Ahmad Wijaya','ahmad.wijaya@email.com','+628123456700','active','2025-06-01'),
  ('m002','ch-garuda','Rina Kusuma','rina.kusuma@email.com','+628123456713','active','2025-06-05'),
  ('m003','ch-garuda','Bayu Setiawan','bayu.setiawan@email.com','+628123456727','active','2024-05-12'),
  ('m004','ch-garuda','Lestari Dewi','lestari.dewi@email.com','+628123456741','active','2026-05-28'),
  ('m005','ch-garuda','Fajar Ramadhan','fajar.ramadhan@email.com','+628123456754','active','2024-06-18'),
  ('m006','ch-garuda','Putri Anggraini','putri.anggraini@email.com','+628123456768','active','2026-06-02'),
  ('m007','ch-magnify','Siti Nurhaliza','siti.nurhaliza@email.com','+628123456782','active','2025-06-15'),
  ('m008','ch-magnify','Andi Pratama','andi.pratama@email.com','+628123456795','active','2024-03-22'),
  ('m009','ch-magnify','Maya Sari','maya.sari@email.com','+628123456809','active','2026-02-10'),
  ('m010','ch-magnify','Reza Mahendra','reza.mahendra@email.com','+628123456823','active','2026-05-20'),
  ('m011','ch-magnify','Indah Permata','indah.permata@email.com','+628123456837','active','2024-06-25'),
  ('m012','ch-magnify','Yoga Pranata','yoga.pranata@email.com','+628123456850','active','2026-06-08'),
  ('m013','ch-amplify','Budi Santoso','budi.santoso@email.com','+628123456864','active','2026-06-01'),
  ('m014','ch-amplify','Citra Lestari','citra.lestari@email.com','+628123456878','active','2026-03-05'),
  ('m015','ch-amplify','Dimas Aryo','dimas.aryo@email.com','+628123456891','active','2024-04-10'),
  ('m016','ch-amplify','Nadia Safira','nadia.safira@email.com','+628123456905','active','2024-05-05'),
  ('m017','ch-amplify','Galih Nugroho','galih.nugroho@email.com','+628123456919','active','2026-04-15'),
  ('m018','ch-amplify','Wulan Maharani','wulan.maharani@email.com','+628123456932','active','2025-06-20'),
  ('m019','ch-rise','Dewi Lestari','dewi.lestari@email.com','+628123456946','active','2026-04-10'),
  ('m020','ch-rise','Hadi Susanto','hadi.susanto@email.com','+628123456960','active','2024-02-14'),
  ('m021','ch-rise','Sinta Permatasari','sinta.permatasari@email.com','+628123456974','active','2025-06-12'),
  ('m022','ch-rise','Eko Prasetyo','eko.prasetyo@email.com','+628123456987','active','2024-06-30'),
  ('m023','ch-rise','Ratna Juwita','ratna.juwita@email.com','+628123457001','active','2026-06-05'),
  ('m024','ch-rise','Bagus Wicaksono','bagus.wicaksono@email.com','+628123457015','active','2026-05-25'),
  ('m025','ch-glorify','Hendra Pratama','hendra.pratama@email.com','+628123457028','active','2024-05-01'),
  ('m026','ch-glorify','Vina Oktaviani','vina.oktaviani@email.com','+628123457042','active','2026-05-08'),
  ('m027','ch-glorify','Rangga Saputra','rangga.saputra@email.com','+628123457056','active','2026-05-15'),
  ('m028','ch-glorify','Tari Melati','tari.melati@email.com','+628123457069','active','2024-07-08'),
  ('m029','ch-glorify','Joko Widodo','joko.widodo@email.com','+628123457083','active','2026-06-10'),
  ('m030','ch-glorify','Ayu Lestari','ayu.lestari@email.com','+628123457097','active','2025-09-02'),
  ('m031','ch-victory','Kadek Surya','kadek.surya@email.com','+628123457111','active','2026-06-03'),
  ('m032','ch-victory','Made Ariani','made.ariani@email.com','+628123457124','active','2024-03-30'),
  ('m033','ch-victory','Wayan Gunawan','wayan.gunawan@email.com','+628123457138','active','2026-05-22'),
  ('m034','ch-victory','Komang Ayu','komang.ayu@email.com','+628123457152','active','2024-06-20'),
  ('m035','ch-victory','Ketut Sariasih','ketut.sariasih@email.com','+628123457165','active','2026-06-12'),
  ('m036','ch-victory','Putu Wirawan','putu.wirawan@email.com','+628123457179','active','2026-06-09')
on conflict (id) do nothing;
