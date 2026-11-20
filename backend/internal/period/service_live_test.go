package period

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// Kelola periode lewat service di atas Postgres sungguhan: id dari rentang
// tanggal, penolakan nama ganda dan rentang beririsan, serta penjaga hapus.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_test
func TestKelolaPeriode(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak diset, integration test dilewati")
	}
	if nama := url[strings.LastIndex(url, "/")+1:]; !strings.Contains(nama, "test") && !strings.Contains(nama, "dev") {
		t.Fatalf("menolak berjalan atas basis data %q: namanya tidak mengandung test/dev", nama)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	t.Cleanup(pool.Close)
	testdb.Serialize(t, pool)

	ctx := scope.WithoutLimit(context.Background())
	const ch = "ch-uji-periode"
	bersih := func() {
		_, _ = pool.Exec(ctx, "DELETE FROM renewal_requests WHERE chapter_id = $1", ch)
		_, _ = pool.Exec(ctx, "DELETE FROM members WHERE chapter_id = $1", ch)
		_, _ = pool.Exec(ctx, "DELETE FROM chapters WHERE id = $1", ch)
		_, _ = pool.Exec(ctx, "DELETE FROM renewal_periods WHERE start_date >= DATE '2090-01-01'")
	}
	bersih()
	t.Cleanup(bersih)

	svc := NewService(NewRepository(pool))
	tanggal := func(s string) *domain.Date {
		d, err := domain.ParseDate(s)
		if err != nil {
			t.Fatal(err)
		}
		return &d
	}
	input := func(nama, mulai, selesai string) domain.SaveRenewalPeriodInput {
		return domain.SaveRenewalPeriodInput{Name: &nama, StartDate: tanggal(mulai), EndDate: tanggal(selesai)}
	}
	status := func(err error) int {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he.Status
		}
		if errors.Is(err, httpx.ErrNotFound) {
			return http.StatusNotFound
		}
		return 0
	}

	s1, err := svc.Create(ctx, input("2090 · Jan–Jun", "2090-01-01", "2090-06-30"))
	if err != nil {
		t.Fatalf("buat semester 1: %v", err)
	}
	if s1.ID != "2090-S1" {
		t.Errorf("id semester 1: dapat %q, mau 2090-S1", s1.ID)
	}
	kuartal, err := svc.Create(ctx, input("2090 · Kuartal 3", "2090-07-01", "2090-09-30"))
	if err != nil {
		t.Fatalf("buat kuartal: %v", err)
	}
	if kuartal.ID != "20900701-20900930" {
		t.Errorf("id kuartal: dapat %q", kuartal.ID)
	}

	t.Run("nama ganda ditolak tanpa membedakan huruf", func(t *testing.T) {
		_, err := svc.Create(ctx, input("2090 · JAN–JUN", "2091-01-01", "2091-06-30"))
		if status(err) != http.StatusConflict || !strings.Contains(err.Error(), "nama") {
			t.Errorf("harus 409 karena nama, dapat %v", err)
		}
	})

	t.Run("rentang beririsan ditolak", func(t *testing.T) {
		_, err := svc.Create(ctx, input("2090 · Jul–Des", "2090-07-01", "2090-12-31"))
		if status(err) != http.StatusConflict || !strings.Contains(err.Error(), "Kuartal 3") {
			t.Errorf("harus 409 menyebut kuartal, dapat %v", err)
		}
	})

	t.Run("ubah nama dan tanggal, id tetap", func(t *testing.T) {
		nama := "2090 · Juli sampai November"
		p, err := svc.Update(ctx, kuartal.ID, domain.SaveRenewalPeriodInput{Name: &nama, EndDate: tanggal("2090-11-30")})
		if err != nil {
			t.Fatalf("ubah: %v", err)
		}
		if p.ID != kuartal.ID || p.Name != nama || p.EndDate.String() != "2090-11-30" {
			t.Errorf("hasil ubah: %+v", p)
		}
		_, err = svc.Update(ctx, kuartal.ID, domain.SaveRenewalPeriodInput{EndDate: tanggal("2090-06-30")})
		if status(err) != http.StatusBadRequest {
			t.Errorf("selesai sebelum mulai harus 400, dapat %v", err)
		}
		_, err = svc.Update(ctx, kuartal.ID, domain.SaveRenewalPeriodInput{StartDate: tanggal("2090-06-01")})
		if status(err) != http.StatusConflict {
			t.Errorf("geser ke semester 1 harus 409, dapat %v", err)
		}
	})

	t.Run("periode yang dipakai tidak bisa dihapus", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `INSERT INTO chapters (id, name, display_name) VALUES ($1, 'ujiperiode', 'Uji Periode')`, ch); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO members (id, chapter_id, name, status, joined_date)
			VALUES ('mem-uji-periode', $1, 'Anggota Periode', 'active', CURRENT_DATE)`, ch); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO renewal_requests (member_id, chapter_id, period)
			VALUES ('mem-uji-periode', $1, $2)`, ch, s1.ID); err != nil {
			t.Fatal(err)
		}

		daftar, err := svc.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range daftar {
			if p.ID == s1.ID && p.RequestCount != 1 {
				t.Errorf("requestCount %s: dapat %d, mau 1", p.ID, p.RequestCount)
			}
		}
		// ST chapter lain tidak melihat angka chapter ini.
		lain, err := svc.List(scope.WithChapter(context.Background(), "ch-lain"))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range lain {
			if p.ID == s1.ID && p.RequestCount != 0 {
				t.Errorf("ST chapter lain melihat %d permintaan", p.RequestCount)
			}
		}

		if err := svc.Delete(ctx, s1.ID); status(err) != http.StatusConflict {
			t.Errorf("hapus periode terpakai harus 409, dapat %v", err)
		}
		if err := svc.Delete(ctx, kuartal.ID); err != nil {
			t.Errorf("hapus periode kosong: %v", err)
		}
		if err := svc.Delete(ctx, kuartal.ID); status(err) != http.StatusNotFound {
			t.Errorf("hapus dua kali harus 404, dapat %v", err)
		}
	})

	t.Run("basis data menolak permintaan tanpa periode terdaftar", func(t *testing.T) {
		_, err := pool.Exec(ctx, `INSERT INTO renewal_requests (member_id, chapter_id, period)
			VALUES ('mem-uji-periode', $1, '2090-TIDAK-ADA')`, ch)
		if err == nil || !strings.Contains(err.Error(), "renewal_requests_period_fkey") {
			t.Errorf("harus ditolak foreign key, dapat %v", err)
		}
	})
}
