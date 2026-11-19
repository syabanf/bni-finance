package invoice

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
	"github.com/syabanf/bni-finance/backend/internal/testdb"
)

// aturRincian menyetel pengaturan biaya untuk satu tes dan mengembalikannya
// ke keadaan semula sesudahnya. Kurs 16.000 dan komponen dari lembar klien
// menghasilkan renewal 11.523.200.
func aturRincian(t *testing.T, pool *pgxpool.Pool, aktif bool) {
	t.Helper()
	ctx := context.Background()
	var lamaAktif bool
	var lamaKurs int64
	if err := pool.QueryRow(ctx,
		`SELECT rincian_aktif, usd_rate FROM fee_settings WHERE id = 'default'`).Scan(&lamaAktif, &lamaKurs); err != nil {
		t.Fatalf("baca pengaturan biaya: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE fee_settings SET rincian_aktif = $1, usd_rate = $2 WHERE id = 'default'`, lamaAktif, lamaKurs)
	})
	if _, err := pool.Exec(ctx, `
		UPDATE fee_settings SET rincian_aktif = $1, usd_rate = 16000,
		       komponen_pendaftaran_usd = 180.00, komponen_keanggotaan_usd = 648.83,
		       ppn_persen = 11.00, biaya_training = 500000
		WHERE id = 'default'`, aktif); err != nil {
		t.Fatalf("atur rincian: %v", err)
	}
}

const renewalBerincian = 11_523_200

// Invoice yang nominalnya sama dengan harga berlaku menyimpan rinciannya, dan
// rincian itu tidak ikut berubah saat harga diubah sesudahnya.
func TestInvoiceMenyimpanRincianSaatDibuat(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	aturRincian(t, pool, true)

	chA, _, memA, _ := duaChapter(t, repo)
	ctx := scope.WithoutLimit(context.Background())

	inv, err := repo.Create(ctx, contohInput(memA, chA, renewalBerincian), "", "IDR")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(inv.Rincian) != 2 {
		t.Fatalf("rincian = %+v, ingin Membership Fee dan VAT", inv.Rincian)
	}
	if inv.Rincian[0].Kode != domain.BiayaKeanggotaan || inv.Rincian[0].Jumlah != 10_381_280 ||
		inv.Rincian[1].Kode != domain.BiayaPPN || inv.Rincian[1].Jumlah != 1_141_920 {
		t.Errorf("isi rincian: %+v", inv.Rincian)
	}
	if domain.JumlahBaris(inv.Rincian) != inv.Amount {
		t.Errorf("jumlah rincian %d tidak sama dengan nominal %d", domain.JumlahBaris(inv.Rincian), inv.Amount)
	}

	// Harga naik sesudahnya. Invoice yang sudah ada tetap membawa rincian lama.
	if _, err := pool.Exec(ctx, `UPDATE fee_settings SET usd_rate = 17000 WHERE id = 'default'`); err != nil {
		t.Fatalf("ubah kurs: %v", err)
	}
	ulang, err := repo.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(ulang.Rincian) != 2 || ulang.Rincian[0].Jumlah != 10_381_280 {
		t.Errorf("rincian invoice lama ikut berubah: %+v", ulang.Rincian)
	}
}

// Nominal yang diketik sendiri tidak punya rincian yang sah, dan mengubah
// nominal invoice membuang rincian yang tidak lagi berjumlah sama.
func TestNominalLainTidakMembawaRincian(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	aturRincian(t, pool, true)

	chA, _, memA, _ := duaChapter(t, repo)
	ctx := scope.WithoutLimit(context.Background())

	bebas, err := repo.Create(ctx, contohInput(memA, chA, 5_000_000), "", "IDR")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if bebas.Rincian != nil {
		t.Errorf("nominal bebas membawa rincian: %+v", bebas.Rincian)
	}

	// Diubah ke harga berlaku: rinciannya muncul. Diubah lagi: hilang.
	harga, lain := int64(renewalBerincian), int64(7_000_000)
	diubah, err := repo.Update(ctx, bebas.ID, domain.UpdateInvoiceInput{Amount: &harga})
	if err != nil {
		t.Fatalf("Update ke harga berlaku: %v", err)
	}
	if len(diubah.Rincian) != 2 {
		t.Errorf("setelah nominal disamakan, rincian = %+v", diubah.Rincian)
	}
	diubah, err = repo.Update(ctx, bebas.ID, domain.UpdateInvoiceInput{Amount: &lain})
	if err != nil {
		t.Fatalf("Update ke nominal lain: %v", err)
	}
	if diubah.Rincian != nil {
		t.Errorf("rincian lama tertinggal pada nominal baru: %+v", diubah.Rincian)
	}
}

// Selama rincian belum diaktifkan, invoice dibuat satu baris seperti sebelumnya.
func TestRincianMatiInvoiceSatuBaris(t *testing.T) {
	pool := livePool(t)
	testdb.Serialize(t, pool)
	repo := NewRepository(pool)
	aturRincian(t, pool, false)

	chA, _, memA, _ := duaChapter(t, repo)
	inv, err := repo.Create(scope.WithoutLimit(context.Background()), contohInput(memA, chA, renewalBerincian), "", "IDR")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inv.Rincian != nil {
		t.Errorf("rincian mati tetapi invoice membawa %+v", inv.Rincian)
	}
}
