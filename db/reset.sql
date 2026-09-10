-- Mengosongkan TABEL BISNIS untuk menyiapkan lingkungan lokal dari nol.
--
-- Dipakai `make db-reset`, dan diuji backend/internal/testdb/reset_test.go —
-- berkas terpisah justru supaya tesnya menjalankan SQL YANG SAMA. Kalau SQL ini
-- tinggal di dalam Makefile, tesnya harus menyalinnya, dan salinan itu akan
-- menyimpang tanpa ada yang tahu.
--
-- CASCADE SENGAJA TIDAK DIPAKAI.
--
-- Versi sebelumnya memakai `truncate … cascade` atas daftar yang kurang dua
-- tabel, lalu mengandalkan CASCADE menambal sisanya. CASCADE memang menambal —
-- tapi ia juga menyeret `users`, karena users.chapter_id menunjuk ke chapters.
-- Postgres mengatakannya terang-terangan: "truncate cascades to table users".
--
-- Akibatnya perintah untuk MENYIAPKAN lingkungan lokal justru menghapus setiap
-- akun, sementara pesannya sendiri berkata "users tidak disentuh". Yang
-- menjalankannya terkunci di luar aplikasinya, dan pesannya menyatakan
-- sebaliknya.
--
-- Sekarang tiap tabel disebut namanya. Kalau kelak ada tabel baru yang menunjuk
-- ke salah satunya, TRUNCATE akan GAGAL dengan jelas alih-alih diam-diam ikut
-- mengosongkannya — gagal berisik lebih murah daripada data yang hilang tanpa
-- disadari.

-- Pengguna berlingkup ikut hilang bersama chapternya.
--
-- Tidak di-NULL-kan, dan itu penting: ChapterScope() memperlakukan chapter_id
-- kosong sebagai "tidak dibatasi", jadi mengosongkannya akan mengubah seorang
-- ST menjadi pengguna nasional yang melihat seluruh chapter. Menghapus barisnya
-- lebih jujur daripada diam-diam memperluas haknya.
delete from users where chapter_id is not null;

truncate invoice_audit_log, payments, invoices, members,
         renewal_requests, reminder_log, fee_settings restart identity;

-- DELETE, bukan TRUNCATE: selama constraint users.chapter_id ada, Postgres
-- menolak TRUNCATE atas chapters — terlepas dari ada tidaknya baris yang
-- menunjuk ke sana.
delete from chapters;
