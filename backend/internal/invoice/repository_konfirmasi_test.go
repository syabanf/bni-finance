package invoice

import (
	"context"
	"strings"
	"testing"

	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// INVOICE RENEWAL BARU TERBIT SETELAH MC MENJAWAB "DITERIMA".
//
// Alurnya Member, lalu Konfirmasi Renewal, baru Invoice. Yang dibaca adalah
// permintaan TERAKHIR member itu: jawaban "Diterima" periode lalu tidak boleh
// membuka tagihan periode ini setelah permintaan baru dibuat dan belum dijawab.
//
//	make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
func TestRenewalMenungguKonfirmasi(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(context.Background())

	chA, _, memA, _ := duaChapter(t, repo)
	// duaChapter sudah mengonfirmasi; mulai dari keadaan tanpa konfirmasi.
	if _, err := pool.Exec(ctx, "DELETE FROM renewal_requests WHERE member_id = $1", memA); err != nil {
		t.Fatalf("bersihkan konfirmasi: %v", err)
	}

	jawab := func(period, answer string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO renewal_requests (member_id, chapter_id, period, answer, requested_at)
			VALUES ($1,$2,$3,$4::renewal_answer, now())
			ON CONFLICT (member_id, period) DO UPDATE SET answer = $4::renewal_answer, requested_at = now()`,
			memA, chA, period, answer); err != nil {
			t.Fatalf("catat jawaban %s: %v", answer, err)
		}
	}
	coba := func(nama, harus string) {
		t.Helper()
		_, err := repo.Create(ctx, contohInput(memA, chA, 1_500_000), "", "IDR")
		if harus == "" {
			if err != nil {
				t.Fatalf("%s: harus diterima, dapat %v", nama, err)
			}
			return
		}
		if httpx.StatusOf(err) != 400 || !strings.Contains(err.Error(), harus) {
			t.Errorf("%s: mau 400 berisi %q, dapat %v", nama, harus, err)
		}
	}

	coba("belum pernah diminta", "belum dimintai konfirmasi")
	jawab("uji-1", "pending")
	coba("belum dijawab", "belum dijawab")
	jawab("uji-1", "will_not")
	coba("ditolak", "tidak memperpanjang")
	jawab("uji-1", "unsure")
	coba("belum pasti", "belum pasti")
	jawab("uji-1", "will_renew")
	coba("diterima", "")
	// Permintaan baru untuk periode berikutnya menutup lagi jalannya.
	jawab("uji-2", "pending")
	coba("periode baru belum dijawab", "belum dijawab")
}
