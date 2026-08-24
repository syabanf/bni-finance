package invoice

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
)

const columns = `
	id, number, member_id, chapter_id, type, amount, currency,
	due_date, period_start, period_end, status,
	paper_id_invoice_id, paper_id_invoice_url, paper_id_payment_url, paper_id_sent_at,
	paper_id_reminder_count,
	payment_provider, xendit_external_id, xendit_payment_id, xendit_payment_method,
	xendit_va_bank, xendit_va_number, xendit_qris_string, xendit_payment_status, xendit_expires_at,
	paid_at, paid_amount, notes, created_by, cancelled_by, cancelled_at, cancel_reason,
	created_at, updated_at`

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type scannable interface {
	Scan(dest ...any) error
}

// salahMasukan menerjemahkan galat basis data yang SEBENARNYA kesalahan klien.
//
// Tanpa ini semuanya keluar sebagai 500 "terjadi kesalahan pada server", dan
// pesan itu menyuruh orang mencari kerusakan server yang tidak pernah ada.
// Dibuktikan dengan dua permintaan yang salahnya jelas di sisi pemanggil:
//
//	memberId yang tidak ada -> 500  (seharusnya 400)
//	amount di luar jangkauan -> 500  (seharusnya 400)
//
// Kelas yang sama pernah menipu pada blackbox, ketika 404 tercatat sebagai 500.
func salahMasukan(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "23503"): // foreign key violation
		switch {
		case strings.Contains(msg, "member_id"):
			return httpx.BadRequest("memberId tidak ditemukan")
		case strings.Contains(msg, "chapter_id"):
			return httpx.BadRequest("chapterId tidak ditemukan")
		}
		return httpx.BadRequest("data rujukan tidak ditemukan")
	case strings.Contains(msg, "23505"): // unique violation
		return httpx.Conflict("nomor invoice tersebut sudah dipakai")
	// Nominal di luar jangkauan kolom. Kolomnya sudah dilebarkan ke bigint,
	// jadi ini kini hanya tercapai oleh angka yang memang mustahil — tapi
	// jawabannya tetap harus 400, bukan 500.
	case strings.Contains(msg, "is greater than maximum value"),
		strings.Contains(msg, "is less than minimum value"),
		strings.Contains(msg, "22003"):
		return httpx.BadRequest("nominal di luar jangkauan yang bisa disimpan")
	}
	return nil
}

// scan maps a row onto the domain model. Date columns come back as time.Time
// and are wrapped so the JSON stays YYYY-MM-DD.
func scan(row scannable) (*domain.Invoice, error) {
	var inv domain.Invoice
	var due, periodStart, periodEnd time.Time

	err := row.Scan(
		&inv.ID, &inv.Number, &inv.MemberID, &inv.ChapterID, &inv.Type, &inv.Amount, &inv.Currency,
		&due, &periodStart, &periodEnd, &inv.Status,
		&inv.PaperIDInvoiceID, &inv.PaperIDInvoiceURL, &inv.PaperIDPaymentURL, &inv.PaperIDSentAt,
		&inv.PaperIDReminderCount,
		&inv.PaymentProvider, &inv.XenditExternalID, &inv.XenditPaymentID, &inv.XenditPaymentMethod,
		&inv.XenditVaBank, &inv.XenditVaNumber, &inv.XenditQrisString, &inv.XenditPaymentStatus, &inv.XenditExpiresAt,
		&inv.PaidAt, &inv.PaidAmount, &inv.Notes, &inv.CreatedBy, &inv.CancelledBy, &inv.CancelledAt, &inv.CancelReason,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		if e := salahMasukan(err); e != nil {
			return nil, e
		}
		return nil, fmt.Errorf("scan invoice: %w", err)
	}

	inv.DueDate = domain.NewDate(due)
	inv.PeriodStart = domain.NewDate(periodStart)
	inv.PeriodEnd = domain.NewDate(periodEnd)
	return &inv, nil
}

// List returns a filtered page plus the total row count for that filter.
func (r *Repository) List(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.Status != "" {
		if f.Status == "outstanding" {
			// Sama seperti UI: outstanding = sent + overdue.
			where = append(where, "status IN ('sent','overdue')")
		} else {
			add("status = $%d", domain.InvoiceStatus(f.Status))
		}
	}
	if f.Type != "" {
		add("type = $%d", domain.InvoiceType(f.Type))
	}
	if f.ChapterID != "" {
		add("chapter_id = $%d", f.ChapterID)
	}
	if f.MemberID != "" {
		add("member_id = $%d", f.MemberID)
	}
	if f.Search != "" {
		add("number ILIKE $%d", "%"+f.Search+"%")
	}
	if f.DueFrom != "" {
		add("due_date >= $%d", f.DueFrom)
	}
	if f.DueTo != "" {
		add("due_date <= $%d", f.DueTo)
	}
	if f.IssuedFrom != "" {
		add("created_at >= $%d", f.IssuedFrom)
	}
	if f.IssuedTo != "" {
		// inclusive: cover the whole end day
		add("created_at < ($%d::date + interval '1 day')", f.IssuedTo)
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM invoices WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung invoice: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		"SELECT %s FROM invoices WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		columns, clause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ambil invoice: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Invoice, 0, f.Limit)
	for rows.Next() {
		inv, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *inv)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Invoice, error) {
	return scan(r.db.QueryRow(ctx, "SELECT "+columns+" FROM invoices WHERE id = $1", id))
}

// Create inserts the invoice and its first audit entry in one transaction, so
// no invoice can exist without a timeline.
// Create menulis invoice beserta baris audit pertamanya dalam satu transaksi.
//
// Bila number kosong, nomornya dibangkitkan DI DALAM transaksi ini, setelah
// mengambil advisory lock per tahun. Itu bukan kehati-hatian berlebihan:
// penomoran menghitung baris yang sudah ada, jadi dua pemanggil yang membaca
// sebelum salah satunya menulis akan menghitung nomor yang sama. Indeks unik
// menolak yang kedua, dan pemanggil menerima 500 tanpa penjelasan.
//
// Terukur sebelum diperbaiki: 24 pembuatan serentak → 20 gagal. Aksi massal
// "buat invoice renewal" menembakkan satu batch dari satu klik, jadi ini bukan
// kasus teoretis.
//
// Lock-nya per tahun dan dilepas saat transaksi selesai, jadi yang terserialkan
// hanya penomoran — bukan seluruh penulisan invoice.
func (r *Repository) Create(ctx context.Context, in domain.CreateInvoiceInput, number, currency string) (*domain.Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	if number == "" {
		year := in.DueDate.Time.Year()
		// hashtext memetakan prefix ke satu int64; lock-nya transaction-scoped.
		if _, err := tx.Exec(ctx,
			"SELECT pg_advisory_xact_lock(hashtext($1))", numberPrefix(year)); err != nil {
			return nil, fmt.Errorf("kunci penomoran: %w", err)
		}
		number, err = nextNumberTx(ctx, tx, year)
		if err != nil {
			return nil, err
		}
	}

	// Chapter invoice HARUS chapter membernya. Keduanya datang terpisah dari
	// klien, dan tanpa pemeriksaan ini keduanya boleh berbeda: pengujian
	// membuat invoice untuk mem-003 (BNI Nusantara) dengan chapterId ch-garuda
	// dan dijawab 201. Tidak ada yang merah, tapi pendapatan chapter jadi salah
	// hitung selamanya — dan salahnya tidak terlihat dari invoice itu sendiri.
	//
	// Diperiksa di sini, bukan di service: hanya di dalam transaksi ini
	// jawabannya masih berlaku saat barisnya benar-benar ditulis.
	var chapterMember string
	switch err := tx.QueryRow(ctx,
		"SELECT chapter_id FROM members WHERE id = $1", in.MemberID,
	).Scan(&chapterMember); {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, httpx.BadRequest("memberId tidak ditemukan")
	case err != nil:
		return nil, fmt.Errorf("periksa chapter member: %w", err)
	case chapterMember != in.ChapterID:
		return nil, httpx.BadRequest(fmt.Sprintf(
			"chapterId %q bukan chapter member tersebut (%q)", in.ChapterID, chapterMember))
	}

	const q = `
		INSERT INTO invoices (number, member_id, chapter_id, type, amount, currency,
		                      due_date, period_start, period_end, status, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'draft',$10,$11)
		RETURNING ` + columns

	inv, err := scan(tx.QueryRow(ctx, q,
		number, in.MemberID, in.ChapterID, in.Type, in.Amount, currency,
		in.DueDate.Time, in.PeriodStart.Time, in.PeriodEnd.Time, in.Notes, in.CreatedBy,
	))
	if err != nil {
		return nil, err
	}

	if err := recordAudit(ctx, tx, auditRow{
		InvoiceID: inv.ID,
		Action:    domain.AuditCreated,
		NewStatus: &inv.Status,
		ActorID:   in.CreatedBy,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("simpan invoice: %w", err)
	}
	return inv, nil
}

// Update writes only the fields present in the patch, and appends an audit row
// describing the change. The row is locked FOR UPDATE first so two concurrent
// patches can't record contradictory before-states.
func (r *Repository) Update(ctx context.Context, id string, in domain.UpdateInvoiceInput) (*domain.Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldStatus domain.InvoiceStatus
	if err := tx.QueryRow(ctx, "SELECT status FROM invoices WHERE id = $1 FOR UPDATE", id).Scan(&oldStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, fmt.Errorf("kunci invoice: %w", err)
	}

	// Pemeriksaan yang mengikat, di bawah lock baris. Service sudah memeriksa
	// hal yang sama sebelum transaksi, tetapi status bisa berubah di antaranya.
	if err := oldStatus.ValidateUpdateFrom(in); err != nil {
		return nil, httpx.Conflict(err.Error())
	}

	sets := []string{"updated_at = now()"}
	args := []any{}
	set := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.Amount != nil {
		set("amount", *in.Amount)
	}
	if in.DueDate != nil {
		set("due_date", in.DueDate.Time)
	}
	if in.PeriodStart != nil {
		set("period_start", in.PeriodStart.Time)
	}
	if in.PeriodEnd != nil {
		set("period_end", in.PeriodEnd.Time)
	}
	if in.Notes != nil {
		set("notes", *in.Notes)
	}
	if in.Status != nil {
		set("status", *in.Status)
		switch *in.Status {
		case domain.StatusPaid:
			sets = append(sets, "paid_at = COALESCE(paid_at, now())", "paid_amount = COALESCE(paid_amount, amount)")
		case domain.StatusCancelled:
			sets = append(sets, "cancelled_at = COALESCE(cancelled_at, now())")
		}
	}
	if in.CancelReason != nil {
		set("cancel_reason", *in.CancelReason)
	}
	if in.CancelledBy != nil {
		set("cancelled_by", *in.CancelledBy)
	}
	if in.PaperIDInvoiceID != nil {
		set("paper_id_invoice_id", *in.PaperIDInvoiceID)
	}
	if in.PaperIDInvoiceURL != nil {
		set("paper_id_invoice_url", *in.PaperIDInvoiceURL)
	}
	if in.PaperIDPaymentURL != nil {
		set("paper_id_payment_url", *in.PaperIDPaymentURL)
	}
	if in.PaperIDSentAt != nil {
		set("paper_id_sent_at", *in.PaperIDSentAt)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE invoices SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), columns)

	inv, err := scan(tx.QueryRow(ctx, q, args...))
	if err != nil {
		return nil, err
	}

	entry := auditRow{
		InvoiceID: inv.ID,
		Action:    domain.AuditUpdated,
		ActorID:   in.ActorID,
		ActorName: in.ActorName,
		Notes:     in.CancelReason,
	}
	if in.Status != nil && *in.Status != oldStatus {
		entry.Action = domain.ActionForStatus(*in.Status)
		entry.OldStatus = &oldStatus
		entry.NewStatus = in.Status
	}
	if err := recordAudit(ctx, tx, entry); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("simpan perubahan invoice: %w", err)
	}
	return inv, nil
}

// auditRow is the payload of one invoice_audit_log insert.
type auditRow struct {
	InvoiceID string
	Action    domain.AuditAction
	OldStatus *domain.InvoiceStatus
	NewStatus *domain.InvoiceStatus
	ActorID   *string
	ActorName *string
	Notes     *string
}

// recordAudit appends to the timeline inside the caller's transaction, so the
// log can never drift from the invoice it describes.
func recordAudit(ctx context.Context, tx pgx.Tx, e auditRow) error {
	const q = `
		INSERT INTO invoice_audit_log (invoice_id, action, old_status, new_status, actor_id, actor_name, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`
	if _, err := tx.Exec(ctx, q, e.InvoiceID, e.Action, e.OldStatus, e.NewStatus,
		e.ActorID, e.ActorName, e.Notes); err != nil {
		return fmt.Errorf("catat audit log: %w", err)
	}
	return nil
}

// Delete membuang invoice beserta jejak auditnya, dalam satu transaksi.
//
// Versi sebelumnya hanya menghapus barisnya sendiri, dan itu TIDAK PERNAH
// berhasil sekali pun: setiap invoice punya baris invoice_audit_log sejak
// detik ia dibuat — recordAudit menulisnya di dalam transaksi Create — dan
// FK-nya tidak punya klausa on delete. Jadi setiap penghapusan menabrak
//
//	violates foreign key constraint "invoice_audit_log_invoice_id_fkey"
//
// dan keluar sebagai 500. Bukan sebagian: SETIAP invoice, termasuk yang dari
// data contoh, karena semuanya punya jejak audit.
//
// Jejaknya ikut dihapus, bukan disisakan: ia menggambarkan invoice yang sudah
// tidak ada, dan penghapusan hanya diizinkan saat belum ada pembayaran sama
// sekali — dijaga di service lewat CountPayments.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM invoice_audit_log WHERE invoice_id = $1", id); err != nil {
		return fmt.Errorf("hapus jejak audit: %w", err)
	}
	tag, err := tx.Exec(ctx, "DELETE FROM invoices WHERE id = $1", id)
	if err != nil {
		if e := salahMasukan(err); e != nil {
			return e
		}
		return fmt.Errorf("hapus invoice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("hapus invoice: %w", err)
	}
	return nil
}

// CountPayments reports how many payments reference an invoice — used to block
// deletes that would orphan payment rows.
func (r *Repository) CountPayments(ctx context.Context, invoiceID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM payments WHERE invoice_id = $1", invoiceID).Scan(&n)
	return n, err
}

func numberPrefix(year int) string { return fmt.Sprintf("INV-%d-", year) }

// NextNumber memperkirakan nomor berikutnya untuk ditampilkan di form.
//
// HANYA untuk pratinjau. Nomor yang benar-benar dipakai dibangkitkan ulang di
// dalam transaksi Create, di bawah advisory lock — nilai dari sini bisa basi
// begitu invoice lain dibuat, dan memakainya sebagai nomor final adalah balapan
// yang sudah pernah menjatuhkan 20 dari 24 permintaan serentak.
func (r *Repository) NextNumber(ctx context.Context, year int) (string, error) {
	return nextNumberTx(ctx, r.db, year)
}

// querier mencakup pool maupun transaksi, supaya penomoran bisa dipanggil dari
// dalam Create tanpa menyalin SQL-nya.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func nextNumberTx(ctx context.Context, q querier, year int) (string, error) {
	prefix := numberPrefix(year)
	var count int
	if err := q.QueryRow(ctx,
		"SELECT COUNT(*) FROM invoices WHERE number LIKE $1", prefix+"%").Scan(&count); err != nil {
		return "", fmt.Errorf("hitung nomor invoice: %w", err)
	}
	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}
