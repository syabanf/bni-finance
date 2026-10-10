package invoice

import (
	"context"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// FAKTUR PAJAK HANYA UNTUK INVOICE LUNAS, DAN SETIAP LAMPIRAN TERCATAT.
//
// Faktur pajak terbit atas pembayaran yang sudah diterima. Melampirkannya ke
// invoice draft atau yang masih menunggu berarti dokumen pajak untuk uang
// yang belum masuk. ST chapter lain tidak boleh tahu invoice itu ada: 404.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func TestFakturPajakHanyaUntukInvoiceLunas(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(context.Background())

	chA, chB, memA, _ := duaChapter(t, repo)
	inv, err := repo.Create(ctx, contohInput(memA, chA, 1_500_000), "", "IDR")
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, inv.ID) })

	if _, err := repo.AttachTaxInvoice(ctx, inv.ID, "/uploads/faktur-1.pdf", nil, nil); httpx.StatusOf(err) != 400 {
		t.Fatalf("invoice draft harus ditolak 400, dapat %v", err)
	}

	// Draft tidak boleh langsung lunas; jalannya lewat terbit (sent) dulu.
	for _, st := range []domain.InvoiceStatus{domain.StatusSent, domain.StatusPaid} {
		if _, err := repo.Update(ctx, inv.ID, domain.UpdateInvoiceInput{Status: &st}); err != nil {
			t.Fatalf("ubah status ke %s: %v", st, err)
		}
		if st == domain.StatusSent {
			if _, err := repo.AttachTaxInvoice(ctx, inv.ID, "/uploads/faktur-1.pdf", nil, nil); httpx.StatusOf(err) != 400 {
				t.Fatalf("invoice terbit yang belum lunas harus ditolak 400, dapat %v", err)
			}
		}
	}

	nama := "Admin Uji"
	got, err := repo.AttachTaxInvoice(ctx, inv.ID, "/uploads/faktur-1.pdf", nil, &nama)
	if err != nil {
		t.Fatalf("lampirkan: %v", err)
	}
	if got.TaxInvoiceURL == nil || *got.TaxInvoiceURL != "/uploads/faktur-1.pdf" || got.TaxInvoiceAt == nil {
		t.Errorf("faktur tidak tersimpan: url=%v at=%v", got.TaxInvoiceURL, got.TaxInvoiceAt)
	}
	if _, err := repo.AttachTaxInvoice(ctx, inv.ID, "/uploads/faktur-2.pdf", nil, &nama); err != nil {
		t.Fatalf("ganti: %v", err)
	}

	var catatan []string
	rows, err := pool.Query(ctx,
		"SELECT notes FROM invoice_audit_log WHERE invoice_id = $1 AND action = 'tax_invoice' ORDER BY created_at", inv.ID)
	if err != nil {
		t.Fatalf("baca audit: %v", err)
	}
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		catatan = append(catatan, n)
	}
	rows.Close()
	if len(catatan) != 2 || catatan[0] != "faktur pajak dilampirkan" || catatan[1] != "faktur pajak diganti" {
		t.Errorf("audit faktur pajak = %v", catatan)
	}

	stB := scope.WithChapter(context.Background(), chB)
	if _, err := repo.AttachTaxInvoice(stB, inv.ID, "/uploads/x.pdf", nil, nil); httpx.StatusOf(err) != 404 {
		t.Errorf("ST chapter lain harus 404, dapat %v", err)
	}
}
