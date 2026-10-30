package invoice

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
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
	// Satu jawaban "Diterima" membuka satu invoice, bukan dua.
	coba("invoice kedua", "sudah punya invoice renewal")
	// Invoice yang dibatalkan tidak dihitung, jadi penggantinya boleh terbit.
	if _, err := pool.Exec(ctx, "UPDATE invoices SET status = 'cancelled' WHERE member_id = $1", memA); err != nil {
		t.Fatalf("batalkan invoice: %v", err)
	}
	coba("setelah dibatalkan", "")
	// Permintaan baru untuk periode berikutnya menutup lagi jalannya.
	jawab("uji-2", "pending")
	coba("periode baru belum dijawab", "belum dijawab")
}

// DUA KLIK SERENTAK TETAP MENERBITKAN SATU INVOICE.
//
// Nomor yang dikirim klien melewati advisory lock penomoran, jadi tanpa kunci
// baris member kedua transaksi sama-sama tidak melihat invoice yang lain dan
// sama-sama lolos. Berlaku untuk renewal per konfirmasi dan pendaftaran.
func TestPembuatanSerentakHanyaSatuYangLolos(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	ctx := scope.WithoutLimit(context.Background())

	chA, _, memA, _ := duaChapter(t, repo)
	const calon = "mem-uji-serentak-calon"
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, chapter_id, name, status, joined_date)
		VALUES ($1,$2,'Calon Serentak','pending'::member_status, CURRENT_DATE)
		ON CONFLICT (id) DO UPDATE SET status = 'pending', chapter_id = $2`, calon, chA); err != nil {
		t.Fatalf("siapkan calon member: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, "DELETE FROM invoice_audit_log WHERE invoice_id IN (SELECT id FROM invoices WHERE member_id = $1)", calon)
		_, _ = pool.Exec(c, "DELETE FROM invoices WHERE member_id = $1", calon)
		_, _ = pool.Exec(c, "DELETE FROM members WHERE id = $1", calon)
	})

	for _, k := range []struct {
		nama, member, harus string
		tipe                domain.InvoiceType
	}{
		{"renewal", memA, "sudah punya invoice renewal", domain.TypeRenewal},
		{"pendaftaran", calon, "sudah punya invoice pendaftaran", domain.TypeRegistration},
	} {
		t.Run(k.nama, func(t *testing.T) {
			const n = 8
			in := contohInput(k.member, chA, 1_500_000)
			in.Type = k.tipe
			hasil := make(chan error, n)
			gate := make(chan struct{})
			for i := 0; i < n; i++ {
				go func(i int) {
					<-gate
					_, err := repo.Create(ctx, in, fmt.Sprintf("SERENTAK-%s-%d-%d", k.nama, time.Now().UnixNano(), i), "IDR")
					hasil <- err
				}(i)
			}
			close(gate)
			lolos := 0
			for i := 0; i < n; i++ {
				switch err := <-hasil; {
				case err == nil:
					lolos++
				case httpx.StatusOf(err) != 400 || !strings.Contains(err.Error(), k.harus):
					t.Errorf("mau 400 berisi %q, dapat %v", k.harus, err)
				}
			}
			if lolos != 1 {
				t.Errorf("%d dari %d pembuatan serentak lolos; harus tepat 1", lolos, n)
			}
		})
	}
}
