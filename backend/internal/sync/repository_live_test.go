package sync

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/importer"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Apply() is pure SQL, so a fake store proves nothing about it. These tests run
// against a real Postgres, gated the same way as the other integration tests:
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func livePool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset — integration test dilewati")
	}
	name := url[strings.LastIndex(url, "/")+1:]
	if q := strings.Index(name, "?"); q >= 0 {
		name = name[:q]
	}
	if !strings.Contains(name, "test") && !strings.Contains(name, "dev") {
		t.Fatalf("TEST_DATABASE_URL menunjuk ke %q — test ini menghapus data", name)
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung ke database: %v", err)
	}
	t.Cleanup(pool.Close)

	// Satu proses tes pada satu waktu — lihat internal/testdb.
	testdb.Serialize(t, pool)

	_, err = pool.Exec(context.Background(),
		"TRUNCATE invoice_audit_log, payments, invoices, members, chapters RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("bersihkan tabel: %v", err)
	}
	return pool
}

func remote(id, chapterID, chapter, name, status string) RemoteMember {
	return RemoteMember{ID: id, ChapterID: chapterID, Chapter: chapter, Name: name, Status: status}
}

func TestLiveApplyUpsertsAndDeactivates(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := t.Context()
	now := time.Now()

	// First run: everything is new.
	first, err := repo.Apply(ctx, []RemoteMember{
		remote("mem-1", "ch-1", "Garuda", "Budi", "active"),
		remote("mem-2", "ch-1", "Garuda", "Siti", "active"),
		remote("mem-3", "ch-2", "Nusantara", "Andi", "pending"),
	}, now)
	if err != nil {
		t.Fatalf("sinkronisasi pertama: %v", err)
	}
	if first.Chapters != 2 || first.Members != 3 || first.Deactivated != 0 {
		t.Fatalf("hasil pertama salah: %+v", first)
	}

	// Second run: mem-2 disappeared upstream, mem-1 was renamed and moved.
	second, err := repo.Apply(ctx, []RemoteMember{
		remote("mem-1", "ch-2", "Nusantara", "Budi Santoso", "active"),
		remote("mem-3", "ch-2", "Nusantara", "Andi", "active"),
		remote("mem-4", "ch-3", "Merdeka", "Rina", "active"),
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("sinkronisasi kedua: %v", err)
	}
	if second.Deactivated != 1 {
		t.Errorf("mem-2 harusnya dinonaktifkan, dapat %d", second.Deactivated)
	}

	// The vanished member is still there, just inactive — never deleted.
	var status, name, chapterID string
	err = pool.QueryRow(ctx, "SELECT status, name, chapter_id FROM members WHERE id = 'mem-2'").
		Scan(&status, &name, &chapterID)
	if err != nil {
		t.Fatalf("mem-2 hilang dari database — seharusnya hanya dinonaktifkan: %v", err)
	}
	if status != "inactive" {
		t.Errorf("mem-2 harusnya inactive, dapat %q", status)
	}

	// The updated member reflects the new values.
	err = pool.QueryRow(ctx, "SELECT status, name, chapter_id FROM members WHERE id = 'mem-1'").
		Scan(&status, &name, &chapterID)
	if err != nil {
		t.Fatalf("ambil mem-1: %v", err)
	}
	if name != "Budi Santoso" || chapterID != "ch-2" {
		t.Errorf("mem-1 tidak diperbarui: name=%q chapter=%q", name, chapterID)
	}

	var chapters int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM chapters").Scan(&chapters)
	if chapters != 3 {
		t.Errorf("harusnya 3 chapter terkumpul lintas dua sinkronisasi, dapat %d", chapters)
	}
}

// A member with an invoice is exactly the case that made deleting unworkable:
// the foreign key would reject it, and the billing history would be lost.
func TestLiveDeactivationSurvivesInvoices(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := t.Context()

	if _, err := repo.Apply(ctx, []RemoteMember{
		remote("mem-1", "ch-1", "Garuda", "Budi", "active"),
	}, time.Now()); err != nil {
		t.Fatalf("sinkronisasi awal: %v", err)
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO invoices (number, member_id, chapter_id, type, amount,
		                      due_date, period_start, period_end, status)
		VALUES ('INV-SYNC-1','mem-1','ch-1','renewal',1500000,
		        CURRENT_DATE, CURRENT_DATE, CURRENT_DATE + 365, 'sent')`)
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}

	// mem-1 vanishes upstream while still holding an invoice.
	result, err := repo.Apply(ctx, []RemoteMember{
		remote("mem-9", "ch-1", "Garuda", "Orang Lain", "active"),
	}, time.Now())
	if err != nil {
		t.Fatalf("sinkronisasi harus berhasil meski member punya invoice: %v", err)
	}
	if result.Deactivated != 1 {
		t.Errorf("harusnya 1 dinonaktifkan, dapat %d", result.Deactivated)
	}

	var invoices int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM invoices WHERE member_id = 'mem-1'").Scan(&invoices)
	if invoices != 1 {
		t.Errorf("riwayat tagihan hilang: %d invoice tersisa", invoices)
	}
}

// A locally edited chapter name must survive a sync — the display name is
// editable in the app, so overwriting it would undo someone's work.
func TestLiveSyncKeepsLocalDisplayName(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := t.Context()

	if _, err := repo.Apply(ctx, []RemoteMember{
		remote("mem-1", "ch-1", "Garuda", "Budi", "active"),
	}, time.Now()); err != nil {
		t.Fatalf("sinkronisasi awal: %v", err)
	}

	if _, err := pool.Exec(ctx,
		"UPDATE chapters SET display_name = 'BNI Garuda Jakarta' WHERE id = 'ch-1'"); err != nil {
		t.Fatalf("ubah nama tampilan: %v", err)
	}

	if _, err := repo.Apply(ctx, []RemoteMember{
		remote("mem-1", "ch-1", "Garuda", "Budi", "active"),
	}, time.Now()); err != nil {
		t.Fatalf("sinkronisasi ulang: %v", err)
	}

	var display, name string
	pool.QueryRow(ctx, "SELECT display_name, name FROM chapters WHERE id = 'ch-1'").Scan(&display, &name)
	if display != "BNI Garuda Jakarta" {
		t.Errorf("nama tampilan lokal tertimpa sinkronisasi: %q", display)
	}
	if name != "Garuda" {
		t.Errorf("nama kanonik harusnya tetap dari BNI VM, dapat %q", name)
	}
}

func strPtr(v string) *string { return &v }

func mustApply(t *testing.T, repo *Repository, rows ...RemoteMember) {
	t.Helper()
	if _, err := repo.Apply(t.Context(), rows, time.Now()); err != nil {
		t.Fatalf("sinkronisasi: %v", err)
	}
}

func statusOf(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var status string
	if err := pool.QueryRow(t.Context(), "SELECT status FROM members WHERE id = $1", id).Scan(&status); err != nil {
		t.Fatalf("ambil status %s: %v", id, err)
	}
	return status
}

// BNI VM hanya tahu member miliknya, jadi ketiadaan di snapshot baru berarti
// sesuatu bagi member yang pernah ia kembalikan. Baris lain masuk lewat jalur
// aslinya di sini, karena jalur itulah yang menentukan isi kolom-kolomnya.
func TestLiveDeactivationOnlyTouchesBNIVMMembers(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(t.Context())

	// Bentuk id BNI VM sama dengan id Tambah Member: keduanya uuid.
	const (
		vmTetap  = "11111111-1111-4111-8111-111111111111"
		vmHilang = "22222222-2222-4222-8222-222222222222"
		bcID     = "bc-garuda-anasthasia-winna"
		kodeID   = "KODE-0042"
	)
	if _, err := repo.Apply(ctx, []RemoteMember{
		remote(vmTetap, "ch-1", "Garuda", "Budi", "active"),
		remote(vmHilang, "ch-1", "Garuda", "Siti", "active"),
	}, time.Now()); err != nil {
		t.Fatalf("sinkronisasi awal: %v", err)
	}

	if err := importer.NewRepository(pool).UpsertMembers(ctx, []importer.MemberRow{
		{ID: bcID, ChapterID: "ch-1", Name: "Anasthasia Winna", Status: "active"},
		{ID: kodeID, ChapterID: "ch-1", Name: "Dari Berkas Umum", Status: "pending"},
	}); err != nil {
		t.Fatalf("impor: %v", err)
	}

	members := member.NewRepository(pool)
	baru := domain.MemberNewMember
	tambahan, err := members.Create(ctx, domain.CreateMemberInput{
		ChapterID: "ch-1", Name: "Dari Tambah Member", Status: &baru,
	})
	if err != nil {
		t.Fatalf("tambah member: %v", err)
	}
	// PATCH ikut mengisi synced_at, dan itu tidak menjadikannya milik BNI VM.
	telepon := "081234567890"
	if _, err := members.Update(ctx, tambahan.ID, domain.UpdateMemberInput{Phone: &telepon}); err != nil {
		t.Fatalf("ubah member: %v", err)
	}

	result, err := repo.Apply(ctx, []RemoteMember{
		remote(vmTetap, "ch-1", "Garuda", "Budi", "active"),
	}, time.Now())
	if err != nil {
		t.Fatalf("sinkronisasi kedua: %v", err)
	}

	if result.Deactivated != 1 {
		t.Errorf("hanya member BNI VM yang hilang yang boleh dinonaktifkan, dapat %d", result.Deactivated)
	}
	for id, want := range map[string]string{
		vmTetap:     "active",
		vmHilang:    "inactive",
		bcID:        "active",
		kodeID:      "pending",
		tambahan.ID: "new_member",
	} {
		if got := statusOf(t, pool, id); got != want {
			t.Errorf("status %s harusnya %q, dapat %q", id, want, got)
		}
	}
}

// Baris yang sudah ada sebelum asalnya dicatat tidak bisa dibedakan, jadi ia
// aman sampai BNI VM mengembalikannya sekali. Sesudah itu ia member BNI VM biasa.
func TestLiveExistingMemberBecomesBNIVMWhenReturned(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := t.Context()
	budi := remote("mem-1", "ch-1", "Garuda", "Budi", "active")

	mustApply(t, repo, budi)
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, chapter_id, name, status) VALUES ('mem-lama', 'ch-1', 'Lama', 'active')`); err != nil {
		t.Fatalf("buat member lama: %v", err)
	}

	mustApply(t, repo, budi)
	if got := statusOf(t, pool, "mem-lama"); got != "active" {
		t.Fatalf("member yang belum pernah dikembalikan BNI VM dinonaktifkan: %q", got)
	}

	mustApply(t, repo, budi, remote("mem-lama", "ch-1", "Garuda", "Lama", "active"))
	mustApply(t, repo, budi)
	if got := statusOf(t, pool, "mem-lama"); got != "inactive" {
		t.Errorf("member yang hilang dari BNI VM harusnya inactive, dapat %q", got)
	}
}

// BNI VM yang tidak mengirim sebuah nilai tidak berarti nilainya sudah tidak ada.
func TestLiveSyncEmptyValuesKeepStoredData(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := t.Context()

	lengkap := remote("mem-1", "ch-1", "Garuda", "Budi", "active")
	lengkap.Email, lengkap.Phone = strPtr("budi@example.com"), strPtr("0811")
	lengkap.Company, lengkap.BusinessField = strPtr("PT Maju"), strPtr("Konsultan")
	lengkap.JoinedDate, lengkap.RenewalDate = strPtr("2025-01-15"), strPtr("2027-01-15")
	mustApply(t, repo, lengkap)

	sebagian := remote("mem-1", "ch-1", "Garuda", "Budi", "active")
	sebagian.Email, sebagian.Phone = nil, strPtr("")
	sebagian.Company, sebagian.BusinessField = strPtr(""), nil
	sebagian.JoinedDate, sebagian.RenewalDate = strPtr(""), nil
	mustApply(t, repo, sebagian)

	var email, phone, company, field, joined, renewal string
	if err := pool.QueryRow(ctx, `
		SELECT email, phone, company, business_field, joined_date::text, renewal_date::text
		FROM members WHERE id = 'mem-1'`).Scan(&email, &phone, &company, &field, &joined, &renewal); err != nil {
		t.Fatalf("nilai tersimpan terhapus: %v", err)
	}
	if email != "budi@example.com" || phone != "0811" || company != "PT Maju" || field != "Konsultan" ||
		joined != "2025-01-15" || renewal != "2027-01-15" {
		t.Errorf("nilai tersimpan berubah: %s %s %s %s %s %s", email, phone, company, field, joined, renewal)
	}
}

// Kontak tetap milik BNI VM: nilai yang ia kirim menimpa suntingan manual.
func TestLiveSyncContactFollowsBNIVM(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(t.Context())

	vm := remote("mem-1", "ch-1", "Garuda", "Budi", "active")
	vm.Email, vm.Phone = strPtr("vm@example.com"), strPtr("0811")
	mustApply(t, repo, vm)
	if _, err := member.NewRepository(pool).Update(ctx, "mem-1", domain.UpdateMemberInput{
		Email: strPtr("manual@example.com"), Phone: strPtr("0899"),
	}); err != nil {
		t.Fatalf("ubah member: %v", err)
	}
	mustApply(t, repo, vm)

	var email, phone string
	if err := pool.QueryRow(ctx, "SELECT email, phone FROM members WHERE id = 'mem-1'").Scan(&email, &phone); err != nil {
		t.Fatalf("ambil kontak: %v", err)
	}
	if email != "vm@example.com" || phone != "0811" {
		t.Errorf("kontak harusnya mengikuti BNI VM, dapat %q %q", email, phone)
	}
}

// Renewal yang lunas memajukan tanggal di sistem ini lebih dulu daripada di BNI
// VM. Sinkronisasi yang memundurkannya membuat member itu jatuh tempo lagi.
func TestLiveSyncNeverMovesRenewalDateBack(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := t.Context()

	syncRenewal := func(tanggal string) string {
		t.Helper()
		m := remote("mem-1", "ch-1", "Garuda", "Budi", "active")
		m.RenewalDate = &tanggal
		mustApply(t, repo, m)
		var renewal string
		if err := pool.QueryRow(ctx, "SELECT renewal_date::text FROM members WHERE id = 'mem-1'").Scan(&renewal); err != nil {
			t.Fatalf("ambil renewal_date: %v", err)
		}
		return renewal
	}
	syncRenewal("2026-06-01")

	var invoiceID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO invoices (number, member_id, chapter_id, type, amount,
		                      due_date, period_start, period_end, status)
		VALUES ('INV-SYNC-2','mem-1','ch-1','renewal',1500000,
		        CURRENT_DATE, '2027-06-01', '2028-06-01', 'paid')
		RETURNING id`).Scan(&invoiceID); err != nil {
		t.Fatalf("buat invoice: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("mulai transaksi: %v", err)
	}
	defer tx.Rollback(ctx)
	if err := member.NaikkanStatusSetelahLunas(ctx, tx, invoiceID); err != nil {
		t.Fatalf("pelunasan: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("simpan pelunasan: %v", err)
	}

	if got := syncRenewal("2026-06-01"); got != "2028-06-01" {
		t.Errorf("sinkronisasi memundurkan renewal_date yang sudah lunas ke %s", got)
	}
	if got := syncRenewal("2029-06-01"); got != "2029-06-01" {
		t.Errorf("tanggal BNI VM yang lebih jauh harusnya dipakai, dapat %s", got)
	}
}

// Status dari BNI VM hanya dipakai saat baris dibuat. Sesudah itu statusnya
// berubah lewat PATCH, pelunasan, atau hilangnya member dari snapshot.
func TestLiveSyncKeepsLocalStatus(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(t.Context())
	members := member.NewRepository(pool)

	setStatus := func(id string, status domain.MemberStatus) {
		t.Helper()
		if _, err := members.Update(ctx, id, domain.UpdateMemberInput{Status: &status}); err != nil {
			t.Fatalf("ubah status %s: %v", id, err)
		}
	}

	baru := remote("mem-baru", "ch-1", "Garuda", "Baru", "active")
	manual := remote("mem-manual", "ch-1", "Garuda", "Manual", "active")
	aktif := remote("mem-aktif", "ch-1", "Garuda", "Aktif", "active")
	kembali := remote("mem-kembali", "ch-1", "Garuda", "Kembali", "active")
	asing := remote("mem-asing", "ch-1", "Garuda", "Asing", "suspended")
	tunggu := remote("mem-tunggu", "ch-1", "Garuda", "Tunggu", "pending")
	mustApply(t, repo, baru, manual, aktif, kembali, asing, tunggu)

	// Baris baru mengambil status BNI VM, dan yang tak dikenal menjadi active.
	for id, want := range map[string]string{"mem-asing": "active", "mem-tunggu": "pending"} {
		if got := statusOf(t, pool, id); got != want {
			t.Errorf("status awal %s harusnya %q, dapat %q", id, want, got)
		}
	}

	setStatus("mem-baru", domain.MemberNewMember)
	setStatus("mem-manual", domain.MemberInactive)
	aktif.Status = "inactive"
	mustApply(t, repo, baru, manual, aktif, asing, tunggu) // mem-kembali hilang dari snapshot
	if got := statusOf(t, pool, "mem-kembali"); got != "inactive" {
		t.Fatalf("member yang hilang harusnya inactive, dapat %q", got)
	}
	mustApply(t, repo, baru, manual, aktif, kembali, asing, tunggu)

	for id, want := range map[string]string{
		"mem-baru":    "new_member",
		"mem-manual":  "inactive",
		"mem-aktif":   "active",
		"mem-kembali": "inactive",
	} {
		if got := statusOf(t, pool, id); got != want {
			t.Errorf("status %s harusnya %q, dapat %q", id, want, got)
		}
	}
}
