// Package dashboard serves the aggregated KPI payload the dashboard and report
// pages render. It is read-only and computed entirely in SQL.
package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// TrendWindowDays is the length of the comparison window behind every trend
// figure: the last N days against the N days before them.
const TrendWindowDays = 30

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// totals is the flat shape the aggregate query returns before it is folded
// into the nested DashboardSummary.
type totals struct {
	totalCount, totalAmount             int64
	paidCount, paidAmount               int64
	outstandingCount, outstandingAmount int64
	overdueCount, overdueAmount         int64

	curTotal, prevTotal             int
	curPaid, prevPaid               int
	curOutstanding, prevOutstanding int
	curOverdue, prevOverdue         int
}

// One pass over `invoices` produces both the absolute figures and the two
// comparison windows — cheaper and more consistent than eight separate queries.
//
// Windows use make_interval(days => $1::int) rather than ($1 || ' days')::interval:
// the concatenated form makes Postgres infer $1 as TEXT, which then rejects the
// int the driver sends.
// FILTER attaches to the aggregate itself — coalesce(sum(x),0) FILTER (...) is
// a syntax error, so the coalesce has to wrap the filtered aggregate instead.
const totalsQuery = `
SELECT
  count(*) FILTER (WHERE status <> 'cancelled'),
  coalesce(sum(amount) FILTER (WHERE status <> 'cancelled'), 0),
  count(*) FILTER (WHERE status = 'paid'),
  coalesce(sum(coalesce(paid_amount, amount)) FILTER (WHERE status = 'paid'), 0),
  count(*) FILTER (WHERE status IN ('sent','overdue')),
  coalesce(sum(amount) FILTER (WHERE status IN ('sent','overdue')), 0),
  count(*) FILTER (WHERE status = 'overdue'),
  coalesce(sum(amount) FILTER (WHERE status = 'overdue'), 0),

  count(*) FILTER (WHERE status <> 'cancelled' AND created_at >= now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status <> 'cancelled' AND created_at >= now() - make_interval(days => 2 * $1::int)
                                              AND created_at <  now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status = 'paid' AND paid_at >= now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status = 'paid' AND paid_at >= now() - make_interval(days => 2 * $1::int)
                                        AND paid_at <  now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status IN ('sent','overdue') AND created_at >= now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status IN ('sent','overdue') AND created_at >= now() - make_interval(days => 2 * $1::int)
                                                     AND created_at <  now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status = 'overdue' AND created_at >= now() - make_interval(days => $1::int)),
  count(*) FILTER (WHERE status = 'overdue' AND created_at >= now() - make_interval(days => 2 * $1::int)
                                           AND created_at <  now() - make_interval(days => $1::int))
FROM invoices`

func (r *Repository) totals(ctx context.Context) (*totals, error) {
	var t totals
	err := r.db.QueryRow(ctx, totalsQuery, TrendWindowDays).Scan(
		&t.totalCount, &t.totalAmount,
		&t.paidCount, &t.paidAmount,
		&t.outstandingCount, &t.outstandingAmount,
		&t.overdueCount, &t.overdueAmount,
		&t.curTotal, &t.prevTotal,
		&t.curPaid, &t.prevPaid,
		&t.curOutstanding, &t.prevOutstanding,
		&t.curOverdue, &t.prevOverdue,
	)
	if err != nil {
		return nil, fmt.Errorf("hitung ringkasan invoice: %w", err)
	}
	return &t, nil
}

// renewalDue counts memberships lapsing in the coming window, compared against
// how many lapsed in the window just gone.
func (r *Repository) renewalDue(ctx context.Context) (current, previous int, err error) {
	const q = `
	SELECT
	  count(*) FILTER (WHERE renewal_date >= CURRENT_DATE
	                     AND renewal_date <= CURRENT_DATE + $1::int),
	  count(*) FILTER (WHERE renewal_date >= CURRENT_DATE - $1::int
	                     AND renewal_date <  CURRENT_DATE)
	FROM members
	WHERE status = 'active' AND renewal_date IS NOT NULL`

	if err = r.db.QueryRow(ctx, q, TrendWindowDays).Scan(&current, &previous); err != nil {
		return 0, 0, fmt.Errorf("hitung jatuh tempo keanggotaan: %w", err)
	}
	return current, previous, nil
}

func (r *Repository) statusBreakdown(ctx context.Context) ([]domain.StatusCount, error) {
	rows, err := r.db.Query(ctx, "SELECT status, count(*) FROM invoices GROUP BY status ORDER BY status")
	if err != nil {
		return nil, fmt.Errorf("hitung sebaran status: %w", err)
	}
	defer rows.Close()

	items := make([]domain.StatusCount, 0, 5)
	for rows.Next() {
		var s domain.StatusCount
		if err := rows.Scan(&s.Status, &s.Count); err != nil {
			return nil, fmt.Errorf("scan sebaran status: %w", err)
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

// monthly returns the last `months` calendar months, including empty ones —
// generate_series keeps gaps in the chart from silently disappearing.
func (r *Repository) monthly(ctx context.Context, months int) ([]domain.MonthlyPoint, error) {
	const q = `
	WITH bulan AS (
	  SELECT date_trunc('month', now()) - make_interval(months => n) AS m
	  FROM generate_series($1::int - 1, 0, -1) AS n
	)
	SELECT to_char(m, 'YYYY-MM'),
	  coalesce((SELECT sum(i.amount) FROM invoices i
	             WHERE i.status <> 'cancelled' AND date_trunc('month', i.created_at) = bulan.m), 0),
	  coalesce((SELECT sum(p.amount) FROM payments p
	             WHERE date_trunc('month', p.paid_at) = bulan.m), 0)
	FROM bulan
	ORDER BY m`

	rows, err := r.db.Query(ctx, q, months)
	if err != nil {
		return nil, fmt.Errorf("hitung tren bulanan: %w", err)
	}
	defer rows.Close()

	items := make([]domain.MonthlyPoint, 0, months)
	for rows.Next() {
		var p domain.MonthlyPoint
		if err := rows.Scan(&p.Month, &p.Issued, &p.Paid); err != nil {
			return nil, fmt.Errorf("scan tren bulanan: %w", err)
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// chapterStats keeps chapters with no invoices in the result (LEFT JOIN) so the
// table shows every chapter, not just the billed ones.
func (r *Repository) chapterStats(ctx context.Context) ([]domain.ChapterStat, error) {
	const q = `
	SELECT c.id, c.display_name,
	  count(i.id) FILTER (WHERE i.status <> 'cancelled'),
	  count(i.id) FILTER (WHERE i.status = 'paid'),
	  count(i.id) FILTER (WHERE i.status IN ('sent','overdue')),
	  count(i.id) FILTER (WHERE i.status = 'overdue'),
	  coalesce(sum(i.amount) FILTER (WHERE i.status <> 'cancelled'), 0)
	FROM chapters c
	LEFT JOIN invoices i ON i.chapter_id = c.id
	GROUP BY c.id, c.display_name
	ORDER BY 7 DESC, c.display_name ASC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("hitung statistik chapter: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ChapterStat, 0, 16)
	for rows.Next() {
		var s domain.ChapterStat
		if err := rows.Scan(&s.ChapterID, &s.ChapterName, &s.Total, &s.Paid,
			&s.Outstanding, &s.Overdue, &s.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan statistik chapter: %w", err)
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

// Summary assembles the whole payload.
func (r *Repository) Summary(ctx context.Context, months int) (*domain.DashboardSummary, error) {
	t, err := r.totals(ctx)
	if err != nil {
		return nil, err
	}
	renewalCur, renewalPrev, err := r.renewalDue(ctx)
	if err != nil {
		return nil, err
	}
	breakdown, err := r.statusBreakdown(ctx)
	if err != nil {
		return nil, err
	}
	trend, err := r.monthly(ctx, months)
	if err != nil {
		return nil, err
	}
	chapters, err := r.chapterStats(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.DashboardSummary{
		Total: domain.AmountBucket{
			Count: int(t.totalCount), Amount: t.totalAmount,
			Trend: domain.PercentChange(t.curTotal, t.prevTotal),
		},
		Paid: domain.AmountBucket{
			Count: int(t.paidCount), Amount: t.paidAmount,
			Trend: domain.PercentChange(t.curPaid, t.prevPaid),
		},
		Outstanding: domain.AmountBucket{
			Count: int(t.outstandingCount), Amount: t.outstandingAmount,
			Trend: domain.PercentChange(t.curOutstanding, t.prevOutstanding),
		},
		Overdue: domain.AmountBucket{
			Count: int(t.overdueCount), Amount: t.overdueAmount,
			Trend: domain.PercentChange(t.curOverdue, t.prevOverdue),
		},
		RenewalDue: domain.CountBucket{
			Count: renewalCur,
			Trend: domain.PercentChange(renewalCur, renewalPrev),
		},
		StatusBreakdown: breakdown,
		Monthly:         trend,
		ChapterStats:    chapters,
	}, nil
}
