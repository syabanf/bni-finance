package paperid

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"

	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// MarkSent and SettleByRef are pure SQL, so the stub store proves nothing about
// them. These run against a real Postgres, gated the same way as the other
// integration tests:
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
		t.Fatalf("sambung: %v", err)
	}
	t.Cleanup(pool.Close)
	// Satu proses tes pada satu waktu — lihat internal/testdb.
	testdb.Serialize(t, pool)
	if _, err := pool.Exec(context.Background(),
		"TRUNCATE invoice_audit_log, payments, invoices, members, chapters RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("bersihkan: %v", err)
	}
	return pool
}

// seedDraft creates a chapter, member, and one draft invoice; returns the id.
func seedDraft(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO chapters (id, name, display_name) VALUES ('ch-1','Garuda','BNI Garuda');
		INSERT INTO members (id, chapter_id, name, email, phone)
		VALUES ('mem-1','ch-1','Budi','budi@example.com','081200000001');`)
	if err != nil {
		t.Fatalf("seed chapter/member: %v", err)
	}
	var id string
	err = pool.QueryRow(ctx, `
		INSERT INTO invoices (number, member_id, chapter_id, type, amount,
		                      due_date, period_start, period_end, status)
		VALUES ('INV-2026-001','mem-1','ch-1','renewal',1500000,
		        CURRENT_DATE, CURRENT_DATE, CURRENT_DATE + 365, 'draft')
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	return id
}

func TestLiveMarkSent(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := context.Background()
	id := seedDraft(t, pool)

	res := CreateResult{
		PaperInvoiceID: "pp-uuid-1",
		PaymentURL:     "https://stg-v2.paper.id/pay",
		InvoicePDFURL:  "https://x/INV.pdf",
	}
	due := time.Now().AddDate(0, 0, 30)
	inv, err := repo.MarkSent(ctx, id, res, due, time.Now(), "Admin")
	if err != nil {
		t.Fatalf("MarkSent: %v", err)
	}
	if inv.Status != domain.StatusSent {
		t.Errorf("status harus sent, dapat %s", inv.Status)
	}
	if inv.PaperIDInvoiceID == nil || *inv.PaperIDInvoiceID != "pp-uuid-1" {
		t.Errorf("paper id tidak tersimpan: %v", inv.PaperIDInvoiceID)
	}
	if inv.PaperIDPaymentURL == nil || *inv.PaperIDPaymentURL != "https://stg-v2.paper.id/pay" {
		t.Errorf("payment url tidak tersimpan: %v", inv.PaperIDPaymentURL)
	}

	// A 'sent' audit row must exist.
	var action string
	if err := pool.QueryRow(ctx,
		"SELECT action FROM invoice_audit_log WHERE invoice_id=$1 AND action='sent'", id).Scan(&action); err != nil {
		t.Errorf("audit 'sent' tidak tercatat: %v", err)
	}

	// Sending again must be refused — it's no longer a draft.
	if _, err := repo.MarkSent(ctx, id, res, due, time.Now(), "Admin"); err == nil {
		t.Error("mengirim invoice yang sudah sent seharusnya gagal")
	}
}

func TestLiveSettleByRefIdempotent(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := context.Background()
	id := seedDraft(t, pool)

	// Move it to sent first (as the send path would).
	if _, err := repo.MarkSent(ctx, id, CreateResult{PaperInvoiceID: "pp-uuid-1"},
		time.Now().AddDate(0, 0, 30), time.Now(), "Admin"); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}

	// First callback settles.
	settled, err := repo.SettleByRef(ctx, "pp-uuid-1", "INV-2026-001", "bank_transfer:bni", "PAID", 1_500_000, time.Now())
	if err != nil || !settled {
		t.Fatalf("callback pertama harus melunasi: settled=%v err=%v", settled, err)
	}

	// Duplicate callback is a no-op — Paper.id retries.
	settled, err = repo.SettleByRef(ctx, "pp-uuid-1", "INV-2026-001", "bank_transfer:bni", "PAID", 1_500_000, time.Now())
	if err != nil || settled {
		t.Fatalf("callback ulang harus no-op: settled=%v err=%v", settled, err)
	}

	// Exactly one payment, invoice paid, exactly one 'paid' audit row.
	var payments, paidAudit int
	var status string
	pool.QueryRow(ctx, "SELECT count(*) FROM payments WHERE invoice_id=$1", id).Scan(&payments)
	pool.QueryRow(ctx, "SELECT status FROM invoices WHERE id=$1", id).Scan(&status)
	pool.QueryRow(ctx, "SELECT count(*) FROM invoice_audit_log WHERE invoice_id=$1 AND action='paid'", id).Scan(&paidAudit)
	if payments != 1 {
		t.Errorf("harus tepat 1 pembayaran, dapat %d", payments)
	}
	if status != "paid" {
		t.Errorf("invoice harus paid, dapat %s", status)
	}
	if paidAudit != 1 {
		t.Errorf("harus tepat 1 audit 'paid', dapat %d", paidAudit)
	}
}

// TestLivePaperIDStaging hits the real Paper.id staging API. Gated on
// PAPER_ID_CLIENT_ID so it only runs when credentials are provided:
//
//	PAPER_ID_CLIENT_ID=… PAPER_ID_CLIENT_SECRET=… go test ./internal/paperid/ -run TestLivePaperIDStaging -v
func TestLivePaperIDStaging(t *testing.T) {
	id := os.Getenv("PAPER_ID_CLIENT_ID")
	secret := os.Getenv("PAPER_ID_CLIENT_SECRET")
	if id == "" || secret == "" {
		t.Skip("PAPER_ID_CLIENT_ID/SECRET tidak diset — smoke test Paper.id dilewati")
	}

	c := NewClient(os.Getenv("PAPER_ID_BASE_URL"), id, secret)

	// Staging berayun lebar — 5 dtk sampai 52 dtk terukur untuk panggilan yang
	// sama — dan 5xx sesaat bukan kabar tentang kode kita. Ulang maksimal tiga
	// kali dengan nomor SEGAR tiap percobaan (timeout yang berhasil di hulu
	// membakar nomor lama). Kegagalan 4xx tetap fatal seketika: itu bug nyata.
	var res *CreateResult
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		res, err = c.CreateInvoice(context.Background(), CreateInput{
			Number:        fmt.Sprintf("INV-GOTEST-%s-%d", time.Now().Format("20060102-150405"), attempt),
			InvoiceDate:   time.Now(),
			DueDate:       time.Now().AddDate(0, 0, 30),
			Amount:        1_500_000,
			ItemName:      "Renewal Keanggotaan BNI Grow",
			ItemDesc:      "Perpanjangan tahunan",
			CustomerID:    "wit-gotest-001",
			CustomerName:  "Budi Santoso",
			CustomerEmail: "budi@example.com",
			CustomerPhone: "081200000001",
			// Never actually message anyone from a test.
			SendEmail: false, SendWhatsApp: false,
		})
		if err == nil {
			break
		}
		var ae *apiError
		if errors.As(err, &ae) && ae.Status < 500 {
			t.Fatalf("Paper.id staging menolak (bukan transien): %v", err)
		}
		t.Logf("PERCOBAAN %d GAGAL TRANSIEN: %v", attempt, err)
	}
	if err != nil {
		t.Fatalf("Paper.id staging create setelah 3 percobaan: %v", err)
	}
	if res.PaperInvoiceID == "" || res.PaymentURL == "" {
		t.Errorf("respons Paper.id tidak lengkap: %+v", res)
	}
	t.Logf("Paper.id staging OK → id=%s payment=%s", res.PaperInvoiceID, res.PaymentURL)
}

// MarkReminded HARUS diuji terhadap Postgres sungguhan, bukan stub.
//
// Versi pertamanya menulis kolom from_status dan to_status, sementara tabelnya
// memakai old_status dan new_status — dan memakai nilai enum 'reminded' yang
// belum ada di audit_action. Dua kesalahan yang MUSTAHIL ketahuan lewat stub
// di memori, karena stub tidak punya skema untuk dilanggar.
//
// Akibatnya bukan sekadar tes merah: panggilan ke Paper.id BERHASIL lebih dulu,
// dokumen pengingat terbentuk dan nomornya terbakar permanen, lalu transaksi
// database gagal dan seluruh catatannya hilang. Member menerima tagihan,
// sistem melaporkan 500, dan operator yang mencoba lagi akan membakar nomor
// berikutnya sambil mengirim pesan kedua.
func TestLiveMarkReminded(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := context.Background()
	id := seedDraft(t, pool)

	// Invoice harus sudah terkirim sebelum bisa diingatkan.
	if _, err := repo.MarkSent(ctx, id, CreateResult{
		PaperInvoiceID: "pp-1", PaymentURL: "https://bayar/1", InvoicePDFURL: "https://pdf/1",
	}, time.Now().AddDate(0, 0, 30), time.Now(), "Admin"); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}

	// Nomor urut dipesan lebih dulu, sama seperti jalur nyata.
	seq, err := repo.ReserveReminder(ctx, id)
	if err != nil {
		t.Fatalf("ReserveReminder: %v", err)
	}
	if seq != 1 {
		t.Errorf("pemesanan pertama = %d, mau 1", seq)
	}

	res := CreateResult{
		PaperInvoiceID: "pp-r1", Number: "INV-UJI-R1",
		PaymentURL: "https://bayar/r1", InvoicePDFURL: "https://pdf/r1",
	}
	inv, err := repo.MarkReminded(ctx, id, res, time.Now(), "Admin")
	if err != nil {
		t.Fatalf("MarkReminded: %v", err)
	}

	// Status TIDAK berubah — pengingat bukan penerbitan ulang.
	if inv.Status != domain.StatusSent {
		t.Errorf("status harus tetap sent, dapat %s", inv.Status)
	}
	if inv.PaperIDReminderCount != 1 {
		t.Errorf("penghitung pengingat = %d, mau 1", inv.PaperIDReminderCount)
	}
	// Tautan diperbarui ke dokumen terbaru: itu yang harus dibagikan ke member.
	if inv.PaperIDPaymentURL == nil || *inv.PaperIDPaymentURL != "https://bayar/r1" {
		t.Errorf("tautan bayar tidak diperbarui: %v", inv.PaperIDPaymentURL)
	}

	// Jejak auditnya ada, dan itu satu-satunya cara menjawab "sudah diingatkan
	// berapa kali" saat member mengeluh.
	var action, note string
	if err := pool.QueryRow(ctx,
		`SELECT action::text, notes FROM invoice_audit_log
		 WHERE invoice_id = $1 AND action = 'reminded' ORDER BY created_at DESC LIMIT 1`,
		id).Scan(&action, &note); err != nil {
		t.Fatalf("baris audit 'reminded' tidak ada: %v", err)
	}
	if !strings.Contains(note, "INV-UJI-R1") {
		t.Errorf("catatan audit harus menyebut nomor dokumennya, dapat %q", note)
	}

	// Pengingat kedua menaikkan penghitung, bukan mengulang dari satu.
	seq2, err := repo.ReserveReminder(ctx, id)
	if err != nil {
		t.Fatalf("ReserveReminder kedua: %v", err)
	}
	if seq2 != 2 {
		t.Fatalf("pemesanan kedua = %d, mau 2", seq2)
	}
	inv2, err := repo.MarkReminded(ctx, id, CreateResult{
		PaperInvoiceID: "pp-r2", Number: "INV-UJI-R2", PaymentURL: "https://bayar/r2",
	}, time.Now(), "Admin")
	if err != nil {
		t.Fatalf("MarkReminded kedua: %v", err)
	}
	if inv2.PaperIDReminderCount != 2 {
		t.Errorf("penghitung pengingat kedua = %d, mau 2", inv2.PaperIDReminderCount)
	}
}

// Pemesanan HARUS mengikat meski langkah sesudahnya gagal.
//
// Inilah alasan penghitung dinaikkan sebelum Paper.id dihubungi, bukan
// sesudahnya. Bila kenaikannya ikut dibatalkan saat pencatatan gagal, percobaan
// berikutnya memakai sufiks yang sama — dan karena dokumennya sudah terlanjur
// ada di Paper.id, sufiks itu ditolak selamanya. Terbukti terjadi pada
// INV-2026-001-R1.
func TestLiveReserveReminderMengikatMeskiGagalSesudahnya(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := context.Background()
	id := seedDraft(t, pool)

	n1, err := repo.ReserveReminder(ctx, id)
	if err != nil {
		t.Fatalf("ReserveReminder: %v", err)
	}

	// Tidak ada MarkReminded sesudahnya — meniru Paper.id berhasil lalu
	// pencatatan gagal, atau proses mati di antaranya.
	n2, err := repo.ReserveReminder(ctx, id)
	if err != nil {
		t.Fatalf("ReserveReminder kedua: %v", err)
	}
	if n2 != n1+1 {
		t.Fatalf("pemesanan berikutnya = %d, mau %d — sufiks yang gagal TIDAK boleh dipakai ulang",
			n2, n1+1)
	}
}
