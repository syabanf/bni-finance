package sync

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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
