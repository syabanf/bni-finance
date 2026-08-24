package reminder

import (
	"context"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/paperid"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

// paperAdapter menyambungkan worker ke paperid.Service.
//
// Ada supaya paket reminder tidak perlu mengimpor seluruh klien Paper.id, dan
// supaya workernya bisa diuji tanpa menyeretnya.
type paperAdapter struct {
	svc interface {
		Remind(ctx context.Context, invoiceID string, opts paperid.SendOptions) (*domain.Invoice, error)
	}
}

// NewPaperPengirim membungkus paperid.Service menjadi Pengirim.
func NewPaperPengirim(svc *paperid.Service) Pengirim { return &paperAdapter{svc: svc} }

func (a *paperAdapter) Remind(ctx context.Context, invoiceID string, opts RemindOptions) error {
	// Worker berjalan TANPA permintaan HTTP, jadi contextnya tidak pernah
	// melewati middleware yang memasang lingkup chapter. Tanpa baris ini
	// scope.Chapter gagal tertutup dan setiap query di dalam Remind
	// mengembalikan nol baris — pengingatnya diam-diam tidak pernah menemukan
	// invoicenya.
	//
	// Dinyatakan EKSPLISIT, bukan dibiarkan kosong: worker memang bekerja
	// lintas chapter, dan itu harus terbaca sebagai keputusan, bukan kelalaian.
	ctx = scope.WithoutLimit(ctx)
	_, err := a.svc.Remind(ctx, invoiceID, paperid.SendOptions{
		Email:    opts.Email,
		WhatsApp: opts.WhatsApp,
	})
	return err
}
