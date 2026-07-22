package domain

import "math"

// AmountBucket is one KPI card: how many, how much, and the change against the
// preceding window of the same length.
type AmountBucket struct {
	Count  int     `json:"count"`
	Amount int64   `json:"amount"`
	Trend  float64 `json:"trend"`
}

// CountBucket is a KPI card without a monetary total.
type CountBucket struct {
	Count int     `json:"count"`
	Trend float64 `json:"trend"`
}

type StatusCount struct {
	Status InvoiceStatus `json:"status"`
	Count  int           `json:"count"`
}

// MonthlyPoint is one column of the issued-vs-paid chart. Month is YYYY-MM.
type MonthlyPoint struct {
	Month  string `json:"month"`
	Issued int64  `json:"issued"`
	Paid   int64  `json:"paid"`
}

type ChapterStat struct {
	ChapterID   string `json:"chapterId"`
	ChapterName string `json:"chapterName"`
	Total       int    `json:"total"`
	Paid        int    `json:"paid"`
	Outstanding int    `json:"outstanding"`
	Overdue     int    `json:"overdue"`
	TotalAmount int64  `json:"totalAmount"`
}

// DashboardSummary matches the frontend's DashboardSummary type so the
// dashboard can read straight from this API.
type DashboardSummary struct {
	Total       AmountBucket `json:"total"`
	Paid        AmountBucket `json:"paid"`
	Outstanding AmountBucket `json:"outstanding"`
	Overdue     AmountBucket `json:"overdue"`
	RenewalDue  CountBucket  `json:"renewalDue"`

	StatusBreakdown []StatusCount  `json:"statusBreakdown"`
	Monthly         []MonthlyPoint `json:"monthly"`
	ChapterStats    []ChapterStat  `json:"chapterStats"`
}

// PercentChange is the trend shown on the KPI cards, rounded to one decimal.
// Growth from nothing is reported as +100% rather than infinity.
func PercentChange(current, previous int) float64 {
	switch {
	case previous == 0 && current == 0:
		return 0
	case previous == 0:
		return 100
	}
	pct := (float64(current) - float64(previous)) / float64(previous) * 100
	return math.Round(pct*10) / 10
}
