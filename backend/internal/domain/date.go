package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// DateLayout is the wire format for date-only columns (due_date, period_*).
const DateLayout = "2006-01-02"

// Date is a date-only value. Postgres `date` columns and the frontend both use
// YYYY-MM-DD, so we keep that exact shape on the wire instead of RFC3339.
type Date struct {
	time.Time
}

func NewDate(t time.Time) Date { return Date{Time: t} }

func ParseDate(s string) (Date, error) {
	t, err := time.Parse(DateLayout, strings.TrimSpace(s))
	if err != nil {
		return Date{}, fmt.Errorf("tanggal harus berformat YYYY-MM-DD: %w", err)
	}
	return Date{Time: t}, nil
}

func (d Date) String() string { return d.Format(DateLayout) }

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format(DateLayout) + `"`), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*d = Date{}
		return nil
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Scan implements sql.Scanner so pgx can read `date` columns into Date.
func (d *Date) Scan(v any) error {
	switch t := v.(type) {
	case nil:
		*d = Date{}
		return nil
	case time.Time:
		*d = Date{Time: t}
		return nil
	case string:
		parsed, err := ParseDate(t)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	default:
		return fmt.Errorf("tidak bisa membaca %T sebagai Date", v)
	}
}

// Value implements driver.Valuer.
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.Format(DateLayout), nil
}

func (d Date) IsZero() bool { return d.Time.IsZero() }
