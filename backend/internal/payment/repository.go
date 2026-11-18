package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

const columns = `
	id, invoice_id, amount, paid_at, payment_method,
	paper_id_payment_id, paper_id_status, xendit_payment_id, xendit_status,
	proof_url, note, created_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

func scan(row scannable) (*domain.Payment, error) {
	var p domain.Payment
	err := row.Scan(
		&p.ID, &p.InvoiceID, &p.Amount, &p.PaidAt, &p.PaymentMethod,
		&p.PaperIDPaymentID, &p.PaperIDStatus, &p.XenditPaymentID, &p.XenditStatus,
		&p.ProofURL, &p.Note, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("scan payment: %w", err)
	}
	return &p, nil
}

// klausaPayment merakit WHERE untuk PaymentFilter.
//
// denganMetode=false membuang saringan metode, untuk daftar metode yang bisa
// dipilih: dengan saringannya menyala, daftar itu menyusut jadi satu pilihan.
//
// Saringan yang menyangkut invoice atau member ditulis sebagai subquery pada
// invoice_id, bukan JOIN: payments tetap satu-satunya tabel di FROM, jadi
// kolom tanpa awalan di query pemanggilnya tidak pernah ambigu, dan baris
// pembayaran tidak bisa tergandakan.
func klausaPayment(ctx context.Context, f domain.PaymentFilter, denganMetode bool) (string, []any) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.InvoiceID != "" {
		add("invoice_id = $%d", f.InvoiceID)
	}
	// payments tidak punya chapter_id sendiri — lingkupnya diwarisi dari invoice
	// induknya.
	if lim := scope.Chapter(ctx); lim.Buntu {
		where = append(where, "1=0")
	} else if lim.Terbatas {
		add("invoice_id IN (SELECT id FROM invoices WHERE chapter_id = $%d)", lim.ChapterID)
	}
	if f.ChapterID != "" {
		add("invoice_id IN (SELECT id FROM invoices WHERE chapter_id = $%d)", f.ChapterID)
	}
	if f.Type != "" {
		add("invoice_id IN (SELECT id FROM invoices WHERE type = $%d)", domain.InvoiceType(f.Type))
	}
	if f.Search != "" {
		// Dua subquery, bukan satu JOIN dengan OR: masing-masing bisa memakai
		// indeks trigramnya sendiri (nomor invoice, nama member). Digabung
		// dengan OR dalam satu JOIN, Postgres memindai seluruh invoice.
		add(`(invoice_id IN (SELECT id FROM invoices WHERE number ILIKE $%[1]d)
			OR invoice_id IN (
				SELECT id FROM invoices WHERE member_id IN (
					SELECT id FROM members WHERE name ILIKE $%[1]d)))`, "%"+f.Search+"%")
	}
	if denganMetode && f.Method != "" {
		add("payment_method = $%d", f.Method)
	}
	if f.PaidFrom != "" {
		add("paid_at >= $%d", f.PaidFrom)
	}
	if f.PaidTo != "" {
		add("paid_at < ($%d::date + interval '1 day')", f.PaidTo)
	}

	return strings.Join(where, " AND "), args
}

// Summary menjumlahkan SELURUH hasil filter, dipecah menurut tipe invoice.
func (r *Repository) Summary(ctx context.Context, f domain.PaymentFilter) (*domain.PaymentSummary, error) {
	clause, args := klausaPayment(ctx, f, true)
	rows, err := r.db.Query(ctx, `
		SELECT i.type, COUNT(*), COALESCE(SUM(p.amount), 0)
		FROM payments p JOIN invoices i ON i.id = p.invoice_id
		WHERE p.id IN (SELECT id FROM payments WHERE `+clause+`)
		GROUP BY i.type`, args...)
	if err != nil {
		return nil, fmt.Errorf("ringkas payment: %w", err)
	}
	defer rows.Close()

	out := &domain.PaymentSummary{ByType: map[string]int64{}}
	for rows.Next() {
		var tipe string
		var n int
		var jumlah int64
		if err := rows.Scan(&tipe, &n, &jumlah); err != nil {
			return nil, fmt.Errorf("scan ringkasan payment: %w", err)
		}
		out.Count += n
		out.Amount += jumlah
		out.ByType[tipe] = jumlah
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	clause, args = klausaPayment(ctx, f, false)
	metode, err := r.db.Query(ctx, `
		SELECT DISTINCT payment_method FROM payments
		WHERE payment_method IS NOT NULL AND payment_method <> '' AND `+clause+`
		ORDER BY payment_method`, args...)
	if err != nil {
		return nil, fmt.Errorf("daftar metode: %w", err)
	}
	defer metode.Close()
	for metode.Next() {
		var m string
		if err := metode.Scan(&m); err != nil {
			return nil, fmt.Errorf("scan metode: %w", err)
		}
		out.Methods = append(out.Methods, m)
	}
	return out, metode.Err()
}

func (r *Repository) List(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	clause, args := klausaPayment(ctx, f, true)

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM payments WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung payment: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		"SELECT %s FROM payments WHERE %s ORDER BY paid_at DESC LIMIT $%d OFFSET $%d",
		columns, clause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil payment: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Payment, 0, f.Limit)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	lim := scope.Chapter(ctx)
	switch {
	case lim.Buntu:
		return nil, httpx.ErrNotFound
	case lim.Terbatas:
		return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM payments WHERE id = $1"+
			" AND invoice_id IN (SELECT id FROM invoices WHERE chapter_id = $2)", id, lim.ChapterID))
	}
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM payments WHERE id = $1", id))
}

// CreateAndSettle inserts the payment and — when settle is true — marks the
// invoice paid in the SAME transaction, so we can never end up with a recorded
// payment against an unpaid invoice.
func (r *Repository) CreateAndSettle(
	ctx context.Context,
	in domain.CreatePaymentInput,
	paidAt time.Time,
	settle bool,
) (*domain.Payment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the invoice row so concurrent settlements can't race.
	var status domain.InvoiceStatus
	err = tx.QueryRow(ctx, "SELECT status FROM invoices WHERE id = $1 FOR UPDATE", in.InvoiceID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("invoice tidak ditemukan")
		}
		return nil, fmt.Errorf("ambil invoice: %w", err)
	}
	if status == domain.StatusCancelled {
		return nil, httpx.Conflict("invoice sudah dibatalkan — tidak bisa menerima pembayaran")
	}

	const insert = `
		INSERT INTO payments (invoice_id, amount, paid_at, payment_method, proof_url, note)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING ` + columns

	p, err := scan(tx.QueryRow(ctx, insert,
		in.InvoiceID, in.Amount, paidAt, in.PaymentMethod, in.ProofURL, in.Note))
	if err != nil {
		return nil, err
	}

	if settle && status != domain.StatusPaid {
		_, err = tx.Exec(ctx, `
			UPDATE invoices
			   SET status = 'paid', paid_at = $2, paid_amount = COALESCE(paid_amount, $3), updated_at = now()
			 WHERE id = $1`, in.InvoiceID, paidAt, in.Amount)
		if err != nil {
			return nil, fmt.Errorf("tandai invoice lunas: %w", err)
		}
		if err := member.NaikkanStatusSetelahLunas(ctx, tx, in.InvoiceID); err != nil {
			return nil, err
		}

		// Settling here bypasses invoice.Repository.Update, so the timeline
		// entry has to be written on this path too — same transaction.
		paid := domain.StatusPaid
		_, err = tx.Exec(ctx, `
			INSERT INTO invoice_audit_log (invoice_id, action, old_status, new_status, notes)
			VALUES ($1, 'paid', $2, $3, $4)`,
			in.InvoiceID, status, paid, in.Note)
		if err != nil {
			return nil, fmt.Errorf("catat audit log: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaksi: %w", err)
	}
	return p, nil
}

func (r *Repository) Update(ctx context.Context, id string, in domain.UpdatePaymentInput) (*domain.Payment, error) {
	sets := []string{}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.Amount != nil {
		set("amount", *in.Amount)
	}
	if in.PaidAt != nil {
		set("paid_at", *in.PaidAt)
	}
	if in.PaymentMethod != nil {
		set("payment_method", *in.PaymentMethod)
	}
	if in.ProofURL != nil {
		set("proof_url", *in.ProofURL)
	}
	if in.Note != nil {
		set("note", *in.Note)
	}
	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE payments SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), columns)
	return scan(r.db.QueryRow(ctx, q, args...))
}

// Delete menolak pembayaran atas invoice yang sudah lunas. Status invoice dan
// masa keanggotaan member sudah maju karena pembayaran itu, dan menghapusnya
// tidak memundurkan keduanya.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	var status domain.InvoiceStatus
	switch err := tx.QueryRow(ctx, `
		SELECT i.status FROM payments p JOIN invoices i ON i.id = p.invoice_id
		WHERE p.id = $1 FOR UPDATE OF p, i`, id).Scan(&status); {
	case errors.Is(err, pgx.ErrNoRows):
		return httpx.ErrNotFound
	case err != nil:
		return fmt.Errorf("baca payment: %w", err)
	case status == domain.StatusPaid:
		return httpx.Conflict("pembayaran atas invoice lunas tidak bisa dihapus; ubah catatannya bila ada yang salah")
	}
	if _, err := tx.Exec(ctx, "DELETE FROM payments WHERE id = $1", id); err != nil {
		return fmt.Errorf("hapus payment: %w", err)
	}
	return tx.Commit(ctx)
}
