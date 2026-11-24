package proof

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
	"github.com/syabanf/bni-finance/backend/internal/payment"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

const columns = `
	p.id, p.invoice_id, p.chapter_id, p.amount, p.paid_at, p.payment_method,
	p.proof_url, p.note, p.status, p.submitted_by, p.submitted_at,
	p.reviewed_by, p.reviewed_at, p.review_note, p.payment_id,
	i.number, i.amount, i.status, m.name, c.display_name`

const from = `
	FROM payment_proofs p
	JOIN invoices i ON i.id = p.invoice_id
	JOIN members m  ON m.id = i.member_id
	JOIN chapters c ON c.id = p.chapter_id`

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func scan(row pgx.Row) (*domain.PaymentProof, error) {
	var p domain.PaymentProof
	var paidAt time.Time
	err := row.Scan(
		&p.ID, &p.InvoiceID, &p.ChapterID, &p.Amount, &paidAt, &p.PaymentMethod,
		&p.ProofURL, &p.Note, &p.Status, &p.SubmittedBy, &p.SubmittedAt,
		&p.ReviewedBy, &p.ReviewedAt, &p.ReviewNote, &p.PaymentID,
		&p.InvoiceNumber, &p.InvoiceAmount, &p.InvoiceStatus, &p.MemberName, &p.ChapterName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan bukti bayar: %w", err)
	}
	p.PaidAt = domain.NewDate(paidAt)
	return &p, nil
}

func (r *Repository) List(ctx context.Context, f domain.ProofFilter) ([]domain.PaymentProof, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if klausa, arg, pakai := scope.Chapter(ctx).SQL("p.chapter_id", len(args)+1); klausa != "" {
		if pakai {
			args = append(args, arg)
		}
		where = append(where, klausa)
	}
	if f.Status != "" {
		add("p.status = $%d", f.Status)
	}
	if f.InvoiceID != "" {
		add("p.invoice_id = $%d::uuid", f.InvoiceID)
	}
	if f.ChapterID != "" {
		add("p.chapter_id = $%d", f.ChapterID)
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+from+" WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung bukti bayar: %w", err)
	}
	args = append(args, f.Limit, f.Offset)
	// Yang menunggu di atas, karena itu pekerjaan finance yang belum selesai.
	q := fmt.Sprintf("SELECT %s %s WHERE %s ORDER BY p.status = 'pending' DESC, p.submitted_at DESC LIMIT $%d OFFSET $%d",
		columns, from, clause, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil bukti bayar: %w", err)
	}
	defer rows.Close()
	out := make([]domain.PaymentProof, 0, f.Limit)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}

// GetByID dibatasi chapter pemanggil. Bukti chapter lain dijawab 404.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.PaymentProof, error) {
	klausa, args, err := satuBaris(ctx, id)
	if err != nil {
		return nil, err
	}
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+from+" WHERE "+klausa, args...))
}

// satuBaris menyusun WHERE untuk satu bukti, berikut batas chapter pemanggil.
func satuBaris(ctx context.Context, id string) (string, []any, error) {
	lim := scope.Chapter(ctx)
	switch {
	case lim.Buntu:
		return "", nil, httpx.ErrNotFound
	case lim.Terbatas:
		return "p.id = $1 AND p.chapter_id = $2", []any{id, lim.ChapterID}, nil
	}
	return "p.id = $1", []any{id}, nil
}

// Create menyimpan bukti dari ST. Invoicenya harus sudah terkirim dan belum
// lunas, dan berada di chapter pemanggil.
func (r *Repository) Create(ctx context.Context, in domain.SubmitProofInput, submittedBy string) (*domain.PaymentProof, error) {
	lim := scope.Chapter(ctx)
	if lim.Buntu {
		return nil, httpx.NotFound("invoice tidak ditemukan")
	}

	var chapterID string
	var status domain.InvoiceStatus
	switch err := r.db.QueryRow(ctx,
		"SELECT chapter_id, status FROM invoices WHERE id = $1", in.InvoiceID).Scan(&chapterID, &status); {
	// Invoice chapter lain dijawab sama dengan yang tidak ada, supaya ST tidak
	// bisa memastikan keberadaan invoice di luar chapternya.
	case errors.Is(err, pgx.ErrNoRows), err == nil && lim.Terbatas && chapterID != lim.ChapterID:
		return nil, httpx.NotFound("invoice tidak ditemukan")
	case err != nil:
		return nil, fmt.Errorf("baca invoice: %w", err)
	case status != domain.StatusSent && status != domain.StatusOverdue:
		return nil, httpx.Conflict(fmt.Sprintf(
			"bukti bayar hanya untuk invoice yang sudah dikirim dan belum lunas (status sekarang %s)", status))
	}

	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO payment_proofs (invoice_id, chapter_id, amount, paid_at, payment_method, proof_url, note, submitted_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text`,
		in.InvoiceID, chapterID, in.Amount, in.PaidAt, in.PaymentMethod, in.ProofURL, in.Note, submittedBy,
	).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return nil, httpx.Conflict("invoice ini sudah punya bukti bayar yang menunggu verifikasi finance")
		}
		return nil, fmt.Errorf("simpan bukti bayar: %w", err)
	}
	return r.GetByID(ctx, id)
}

// Approve mencatat pembayaran dari bukti itu dan menandai invoicenya lunas,
// dalam satu transaksi dengan perubahan status buktinya.
func (r *Repository) Approve(ctx context.Context, id, reviewer string, note *string) (*domain.PaymentProof, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	var in domain.CreatePaymentInput
	var paidAt time.Time
	var status domain.ProofStatus
	var proofURL string
	var invoiceStatus domain.InvoiceStatus
	switch err := tx.QueryRow(ctx, `
		SELECT p.invoice_id::text, p.amount, p.paid_at, p.payment_method, p.proof_url, p.note, p.status, i.status
		FROM payment_proofs p JOIN invoices i ON i.id = p.invoice_id
		WHERE p.id = $1 FOR UPDATE OF p, i`, id,
	).Scan(&in.InvoiceID, &in.Amount, &paidAt, &in.PaymentMethod, &proofURL, &in.Note, &status, &invoiceStatus); {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, httpx.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("kunci bukti bayar: %w", err)
	case status != domain.ProofPending:
		return nil, httpx.Conflict("bukti ini sudah diperiksa")
	// Invoice yang sudah lunas lewat jalur lain, misalnya Paper.id, akan
	// tercatat dibayar dua kali bila bukti ini ikut disetujui.
	case invoiceStatus == domain.StatusPaid:
		return nil, httpx.Conflict("invoice ini sudah lunas; tolak bukti ini bila pembayarannya sama")
	}
	in.ProofURL = &proofURL

	p, err := payment.CatatDalamTx(ctx, tx, in, paidAt, true)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE payment_proofs
		   SET status = 'approved', reviewed_by = $2, reviewed_at = now(), review_note = $3, payment_id = $4
		 WHERE id = $1`, id, reviewer, note, p.ID); err != nil {
		return nil, fmt.Errorf("setujui bukti bayar: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("simpan persetujuan: %w", err)
	}
	return r.GetByID(ctx, id)
}

// Reject menolak bukti yang masih menunggu. Invoicenya tidak berubah, dan ST
// bisa mengirim bukti baru.
func (r *Repository) Reject(ctx context.Context, id, reviewer, note string) (*domain.PaymentProof, error) {
	var status domain.ProofStatus
	err := r.db.QueryRow(ctx, `
		WITH target AS (SELECT id, status FROM payment_proofs WHERE id = $1 FOR UPDATE),
		     ubah AS (UPDATE payment_proofs p
		                 SET status = 'rejected', reviewed_by = $2, reviewed_at = now(), review_note = $3
		                FROM target t WHERE p.id = t.id AND t.status = 'pending')
		SELECT status FROM target`, id, reviewer, note).Scan(&status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, httpx.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("tolak bukti bayar: %w", err)
	case status != domain.ProofPending:
		return nil, httpx.Conflict("bukti ini sudah diperiksa")
	}
	return r.GetByID(ctx, id)
}

// Delete menarik bukti yang belum diperiksa, di dalam chapter pemanggil.
func (r *Repository) Delete(ctx context.Context, id string) error {
	klausa, args, err := satuBaris(ctx, id)
	if err != nil {
		return err
	}
	var status domain.ProofStatus
	err = r.db.QueryRow(ctx, `
		WITH target AS (SELECT p.id, p.status FROM payment_proofs p WHERE `+klausa+` FOR UPDATE),
		     hapus AS (DELETE FROM payment_proofs x USING target t WHERE x.id = t.id AND t.status = 'pending')
		SELECT status FROM target`, args...).Scan(&status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return httpx.ErrNotFound
	case err != nil:
		return fmt.Errorf("tarik bukti bayar: %w", err)
	case status != domain.ProofPending:
		return httpx.Conflict("bukti yang sudah diperiksa finance tidak bisa ditarik")
	}
	return nil
}
