package api_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"

	"github.com/syabanf/bni-finance/backend/internal/auth"
)

// Minimal in-memory stores for the resources added alongside invoices and
// payments. They exist so the route wiring, filters and status codes can be
// exercised without Postgres — not to reimplement its semantics.

// --- chapters ---------------------------------------------------------------

type fakeChapterStore struct {
	mu    sync.RWMutex
	items map[string]domain.Chapter
	seq   int
	// deps lets a test pretend a chapter is still referenced.
	members, invoices int
}

func newFakeChapterStore() *fakeChapterStore {
	return &fakeChapterStore{items: make(map[string]domain.Chapter)}
}

func (s *fakeChapterStore) List(_ context.Context, f domain.ChapterFilter) ([]domain.Chapter, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Chapter, 0, len(s.items))
	for _, c := range s.items {
		if f.CityName != "" && (c.CityName == nil || *c.CityName != f.CityName) {
			continue
		}
		out = append(out, c)
	}
	return out, len(out), nil
}

func (s *fakeChapterStore) GetByID(_ context.Context, id string) (*domain.Chapter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &c, nil
}

func (s *fakeChapterStore) Create(_ context.Context, in domain.CreateChapterInput) (*domain.Chapter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("ch-%03d", s.seq)
	if in.ID != nil && *in.ID != "" {
		id = *in.ID
	}
	if _, exists := s.items[id]; exists {
		return nil, httpx.Conflict("chapter dengan id tersebut sudah ada")
	}
	display := in.Name
	if in.DisplayName != nil && *in.DisplayName != "" {
		display = *in.DisplayName
	}
	c := domain.Chapter{
		ID: id, Name: in.Name, DisplayName: display,
		AreaName: in.AreaName, CityName: in.CityName, SyncedAt: time.Now().UTC(),
	}
	s.items[id] = c
	return &c, nil
}

func (s *fakeChapterStore) Update(_ context.Context, id string, in domain.UpdateChapterInput) (*domain.Chapter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.DisplayName != nil {
		c.DisplayName = *in.DisplayName
	}
	if in.CityName != nil {
		c.CityName = in.CityName
	}
	s.items[id] = c
	return &c, nil
}

func (s *fakeChapterStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return httpx.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *fakeChapterStore) CountDependents(_ context.Context, _ string) (int, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.members, s.invoices, nil
}

// --- members ----------------------------------------------------------------

type fakeMemberStore struct {
	mu       sync.RWMutex
	items    map[string]domain.Member
	seq      int
	invoices int
}

func newFakeMemberStore() *fakeMemberStore {
	return &fakeMemberStore{items: make(map[string]domain.Member)}
}

func (s *fakeMemberStore) List(_ context.Context, f domain.MemberFilter) ([]domain.Member, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Member, 0, len(s.items))
	for _, m := range s.items {
		if f.ChapterID != "" && m.ChapterID != f.ChapterID {
			continue
		}
		if f.Status != "" && string(m.Status) != f.Status {
			continue
		}
		out = append(out, m)
	}
	return out, len(out), nil
}

func (s *fakeMemberStore) GetByID(_ context.Context, id string) (*domain.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &m, nil
}

func (s *fakeMemberStore) RenewalDue(_ context.Context, _, limit int) ([]domain.RenewalDueMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.RenewalDueMember, 0, limit)
	for _, m := range s.items {
		if len(out) == limit {
			break
		}
		if m.Status == domain.MemberActive && m.RenewalDate != nil {
			out = append(out, domain.RenewalDueMember{Member: m, DaysUntilDue: 7})
		}
	}
	return out, nil
}

func (s *fakeMemberStore) Create(_ context.Context, in domain.CreateMemberInput) (*domain.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("mem-%03d", s.seq)
	if in.ID != nil && *in.ID != "" {
		id = *in.ID
	}
	status := domain.MemberActive
	if in.Status != nil {
		status = *in.Status
	}
	m := domain.Member{
		ID: id, ChapterID: in.ChapterID, Name: in.Name, Email: in.Email,
		Phone: in.Phone, Company: in.Company, BusinessField: in.BusinessField,
		Status: status, JoinedDate: in.JoinedDate, RenewalDate: in.RenewalDate,
		SyncedAt: time.Now().UTC(),
	}
	s.items[id] = m
	return &m, nil
}

func (s *fakeMemberStore) Update(_ context.Context, id string, in domain.UpdateMemberInput) (*domain.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	if in.Name != nil {
		m.Name = *in.Name
	}
	if in.Status != nil {
		m.Status = *in.Status
	}
	if in.RenewalDate != nil {
		m.RenewalDate = in.RenewalDate
	}
	s.items[id] = m
	return &m, nil
}

func (s *fakeMemberStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return httpx.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *fakeMemberStore) CountInvoices(_ context.Context, _ string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.invoices, nil
}

// --- settings ---------------------------------------------------------------

type fakeSettingsStore struct {
	mu   sync.RWMutex
	fees domain.FeeSettings
	app  map[string]domain.AppSetting
}

func newFakeSettingsStore() *fakeSettingsStore {
	return &fakeSettingsStore{
		fees: domain.FeeSettings{
			ID: "default", RegistrationFee: 1_500_000, RenewalFee: 1_500_000,
			Currency: "IDR", UpdatedAt: time.Now().UTC(), CreatedAt: time.Now().UTC(),
		},
		app: make(map[string]domain.AppSetting),
	}
}

func (s *fakeSettingsStore) GetFees(context.Context) (*domain.FeeSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f := s.fees
	return &f, nil
}

func (s *fakeSettingsStore) UpdateFees(_ context.Context, in domain.UpdateFeeSettingsInput) (*domain.FeeSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.RegistrationFee != nil {
		s.fees.RegistrationFee = *in.RegistrationFee
	}
	if in.RenewalFee != nil {
		s.fees.RenewalFee = *in.RenewalFee
	}
	if in.Notes != nil {
		s.fees.Notes = in.Notes
	}
	s.fees.UpdatedAt = time.Now().UTC()
	f := s.fees
	return &f, nil
}

func (s *fakeSettingsStore) ListApp(context.Context) ([]domain.AppSetting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.AppSetting, 0, len(s.app))
	for _, v := range s.app {
		out = append(out, v)
	}
	return out, nil
}

func (s *fakeSettingsStore) GetApp(_ context.Context, key string) (*domain.AppSetting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.app[key]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &v, nil
}

func (s *fakeSettingsStore) SetApp(_ context.Context, key, value string) (*domain.AppSetting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	v := domain.AppSetting{Key: key, Value: value, UpdatedAt: &now}
	s.app[key] = v
	return &v, nil
}

func (s *fakeSettingsStore) DeleteApp(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.app[key]; !ok {
		return httpx.ErrNotFound
	}
	delete(s.app, key)
	return nil
}

// --- audit ------------------------------------------------------------------

type fakeAuditStore struct {
	mu      sync.RWMutex
	entries map[string][]domain.AuditEntry
	seq     int
	known   map[string]bool
}

func newFakeAuditStore() *fakeAuditStore {
	return &fakeAuditStore{
		entries: make(map[string][]domain.AuditEntry),
		known:   make(map[string]bool),
	}
}

func (s *fakeAuditStore) ListByInvoice(_ context.Context, invoiceID string, limit int) ([]domain.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.entries[invoiceID]
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (s *fakeAuditStore) Create(_ context.Context, invoiceID string, in domain.CreateAuditEntryInput) (*domain.AuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	action := domain.AuditUpdated
	if in.Action != nil {
		action = *in.Action
	}
	e := domain.AuditEntry{
		ID: fmt.Sprintf("aud-%03d", s.seq), InvoiceID: invoiceID, Action: action,
		ActorID: in.ActorID, ActorName: in.ActorName, Notes: in.Notes,
		CreatedAt: time.Now().UTC(),
	}
	s.entries[invoiceID] = append([]domain.AuditEntry{e}, s.entries[invoiceID]...)
	return &e, nil
}

func (s *fakeAuditStore) InvoiceExists(_ context.Context, invoiceID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.known[invoiceID] {
		return httpx.ErrNotFound
	}
	return nil
}

// --- dashboard --------------------------------------------------------------

type fakeDashboardStore struct{}

func (fakeDashboardStore) Summary(_ context.Context, months int) (*domain.DashboardSummary, error) {
	monthly := make([]domain.MonthlyPoint, months)
	for i := range monthly {
		monthly[i] = domain.MonthlyPoint{Month: fmt.Sprintf("2026-%02d", i+1)}
	}
	return &domain.DashboardSummary{
		Total:           domain.AmountBucket{Count: 3, Amount: 4_500_000, Trend: 50},
		Paid:            domain.AmountBucket{Count: 1, Amount: 1_500_000},
		Outstanding:     domain.AmountBucket{Count: 2, Amount: 3_000_000},
		Overdue:         domain.AmountBucket{},
		RenewalDue:      domain.CountBucket{Count: 4},
		StatusBreakdown: []domain.StatusCount{{Status: domain.StatusDraft, Count: 3}},
		Monthly:         monthly,
		ChapterStats:    []domain.ChapterStat{},
	}, nil
}

// --- users ------------------------------------------------------------------

type fakeUserStore struct {
	mu    sync.RWMutex
	items map[string]domain.User
	seq   int
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{items: make(map[string]domain.User)}
}

func (s *fakeUserStore) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.items {
		if u.Email == domain.NormalizeEmail(email) {
			return &u, nil
		}
	}
	return nil, httpx.ErrNotFound
}

func (s *fakeUserStore) GetByID(_ context.Context, id string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &u, nil
}

func (s *fakeUserStore) List(context.Context) ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.User, 0, len(s.items))
	for _, u := range s.items {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *fakeUserStore) Create(_ context.Context, email, hash, name string, role domain.UserRole, chapterID *string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	email = domain.NormalizeEmail(email)
	for _, u := range s.items {
		if u.Email == email {
			return nil, httpx.Conflict("email tersebut sudah terdaftar")
		}
	}
	s.seq++
	u := domain.User{
		ID: fmt.Sprintf("usr-%03d", s.seq), Email: email, PasswordHash: hash,
		Name: name, Role: role, ChapterID: chapterID,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	s.items[u.ID] = u
	return &u, nil
}

func (s *fakeUserStore) UpdateName(_ context.Context, id, name string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	u.Name = name
	s.items[id] = u
	return &u, nil
}

func (s *fakeUserStore) UpdateRole(_ context.Context, id string, role domain.UserRole) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	u.Role = role
	s.items[id] = u
	return &u, nil
}

func (s *fakeUserStore) UpdatePasswordHash(_ context.Context, id, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.items[id]
	if !ok {
		return httpx.ErrNotFound
	}
	u.PasswordHash = hash
	s.items[id] = u
	return nil
}

func (s *fakeUserStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return httpx.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *fakeUserStore) CountAdmins(context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, u := range s.items {
		if u.Role == domain.RoleAdmin {
			n++
		}
	}
	return n, nil
}

// Varian Guarded meniru kontrak repository sungguhan: pemeriksaan
// admin-terakhir dan penulisannya terjadi di bawah SATU lock, tidak terpisah.
// Kalau fake memisahkannya, tes in-memory akan menguji perilaku yang berbeda
// dari produksi — persis celah yang membuat balapan ini lolos selama ini.
func (s *fakeUserStore) UpdateRoleGuarded(_ context.Context, id string, role domain.UserRole) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.items[id]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	if u.Role == domain.RoleAdmin && role != domain.RoleAdmin && s.countAdminsLocked() <= 1 {
		return nil, auth.ErrLastAdmin
	}
	u.Role = role
	s.items[id] = u
	out := u
	return &out, nil
}

func (s *fakeUserStore) DeleteGuarded(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.items[id]
	if !ok {
		return httpx.ErrNotFound
	}
	if u.Role == domain.RoleAdmin && s.countAdminsLocked() <= 1 {
		return auth.ErrLastAdmin
	}
	delete(s.items, id)
	return nil
}

func (s *fakeUserStore) countAdminsLocked() int {
	n := 0
	for _, u := range s.items {
		if u.Role == domain.RoleAdmin {
			n++
		}
	}
	return n
}
