package paperid

import (
	"context"
	"testing"
	"time"
)

// STATUS MEMBER MAJU SAAT INVOICENYA LUNAS, DI TRANSAKSI YANG SAMA.
//
//	pendaftaran lunas  visitor, pending  ->  new_member
//	renewal lunas      new_member        ->  active
//
// Tipe invoice mengikuti status: pendaftaran hanya untuk yang belum anggota,
// renewal hanya untuk yang sudah. Tanpa aktivasi ini, visitor yang sudah
// membayar tetap visitor, dan tahun depan tidak ada jalan menagihnya renewal.
// renewal_date harus terisi akhir periode invoice, dan tidak boleh mundur bila
// yang tersimpan sudah lebih jauh.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func TestLiveSettleMemajukanStatusMember(t *testing.T) {
	pool := livePool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO chapters (id, name, display_name) VALUES ('ch-1','Garuda','BNI Garuda');
		INSERT INTO members (id, chapter_id, name, status, renewal_date)
		VALUES ('mem-tamu','ch-1','Tamu','visitor', NULL),
		       ('mem-calon','ch-1','Calon','pending', CURRENT_DATE + 900),
		       ('mem-aktif','ch-1','Aktif','active', CURRENT_DATE + 30),
		       ('mem-baru','ch-1','Baru','new_member', CURRENT_DATE + 10);`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	seed := func(nomor, member, tipe, paperID string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO invoices (number, member_id, chapter_id, type, amount,
			                      due_date, period_start, period_end, status, paper_id_invoice_id)
			VALUES ($1,$2,'ch-1',$3::invoice_type,1000000,
			        CURRENT_DATE, CURRENT_DATE, CURRENT_DATE + 365, 'sent', $4)
			RETURNING id`, nomor, member, tipe, paperID).Scan(&id); err != nil {
			t.Fatalf("seed invoice %s: %v", nomor, err)
		}
		return id
	}
	seed("INV-T-1", "mem-tamu", "registration", "pp-tamu")
	seed("INV-T-2", "mem-calon", "registration", "pp-calon")
	seed("INV-T-3", "mem-aktif", "renewal", "pp-aktif")
	seed("INV-T-4", "mem-baru", "renewal", "pp-baru")

	for _, pp := range []string{"pp-tamu", "pp-calon", "pp-aktif", "pp-baru"} {
		if settled, err := repo.SettleByRef(ctx, pp, "", "bank_transfer:bni", "PAID", 1_000_000, time.Now()); err != nil || !settled {
			t.Fatalf("%s: settled=%v err=%v", pp, settled, err)
		}
	}

	baca := func(id string) (status string, hari int) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			"SELECT status::text, coalesce(renewal_date - CURRENT_DATE, -1) FROM members WHERE id = $1", id,
		).Scan(&status, &hari); err != nil {
			t.Fatalf("baca %s: %v", id, err)
		}
		return
	}

	if s, h := baca("mem-tamu"); s != "new_member" || h != 365 {
		t.Errorf("visitor: status=%s renewal_date=+%d hari, mau new_member dan +365", s, h)
	}
	// Tanggal yang sudah lebih jauh tidak dimundurkan.
	if s, h := baca("mem-calon"); s != "new_member" || h != 900 {
		t.Errorf("pending: status=%s renewal_date=+%d hari, mau new_member dan +900 (tidak mundur)", s, h)
	}
	// Renewal tidak menyentuh member: tanggalnya milik BNI.
	if s, h := baca("mem-aktif"); s != "active" || h != 30 {
		t.Errorf("renewal: status=%s renewal_date=+%d hari, mau tetap active dan +30", s, h)
	}
	// Renewal pertama new member menjadikannya active; tanggalnya tetap milik BNI.
	if s, h := baca("mem-baru"); s != "active" || h != 10 {
		t.Errorf("new member renewal: status=%s renewal_date=+%d hari, mau active dan +10", s, h)
	}
}
