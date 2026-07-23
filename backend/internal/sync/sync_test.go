package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// fakeBNIVM stands in for BNI Visitor Management so pagination, auth, and the
// error paths can be exercised without the real upstream.
func fakeBNIVM(t *testing.T, total int, wantToken string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+wantToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit == 0 {
			limit = pageSize
		}

		rows := make([]RemoteMember, 0, limit)
		for i := offset; i < offset+limit && i < total; i++ {
			email := fmt.Sprintf("m%03d@example.com", i)
			rows = append(rows, RemoteMember{
				ID:        fmt.Sprintf("mem-%04d", i),
				ChapterID: fmt.Sprintf("ch-%d", i%3),
				Chapter:   fmt.Sprintf("Chapter %d", i%3),
				Name:      fmt.Sprintf("Anggota %d", i),
				Email:     &email,
				Status:    "active",
			})
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":       rows,
			"pagination": map[string]bool{"hasMore": offset+limit < total},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchMembersPaginates(t *testing.T) {
	// More than one page, and deliberately not a round multiple of pageSize.
	const total = pageSize*2 + 37
	srv := fakeBNIVM(t, total, "token-benar")

	members, err := NewClient(srv.URL, "token-benar").FetchMembers(t.Context())
	if err != nil {
		t.Fatalf("ambil member: %v", err)
	}
	if len(members) != total {
		t.Fatalf("dapat %d member, harusnya %d", len(members), total)
	}
	if members[0].ID != "mem-0000" || members[total-1].ID != fmt.Sprintf("mem-%04d", total-1) {
		t.Errorf("urutan atau isi halaman salah: pertama=%s terakhir=%s",
			members[0].ID, members[total-1].ID)
	}
}

func TestFetchMembersRejectsBadToken(t *testing.T) {
	srv := fakeBNIVM(t, 10, "token-benar")

	_, err := NewClient(srv.URL, "token-salah").FetchMembers(t.Context())
	if err == nil {
		t.Fatal("token salah seharusnya gagal")
	}
	// The message has to point at the fix, not just report a status code.
	if !contains(err.Error(), "bni_vm_token") {
		t.Errorf("pesan galat tidak menyebut cara memperbaikinya: %v", err)
	}
}

func TestDeriveChaptersDeduplicates(t *testing.T) {
	members := []RemoteMember{
		{ID: "a", ChapterID: "ch-1", Chapter: "Garuda"},
		{ID: "b", ChapterID: "ch-1", Chapter: "Garuda"},
		{ID: "c", ChapterID: "ch-2", Chapter: "Nusantara"},
		{ID: "d", ChapterID: "ch-3", Chapter: ""},  // nama kosong → pakai id
		{ID: "e", ChapterID: "", Chapter: "Hantu"}, // tanpa id → dilewati
	}
	got := deriveChapters(members)
	if len(got) != 3 {
		t.Fatalf("dapat %d chapter, harusnya 3: %+v", len(got), got)
	}
	byID := map[string]string{}
	for _, c := range got {
		byID[c.ID] = c.Name
	}
	if byID["ch-1"] != "Garuda" || byID["ch-2"] != "Nusantara" {
		t.Errorf("nama chapter salah: %+v", byID)
	}
	if byID["ch-3"] != "ch-3" {
		t.Errorf("chapter tanpa nama harusnya jatuh ke id, dapat %q", byID["ch-3"])
	}
}

// --- service ----------------------------------------------------------------

type stubStore struct {
	token   string
	applied []RemoteMember
	result  *Result
}

func (s *stubStore) Setting(context.Context, string) (string, error) { return s.token, nil }

func (s *stubStore) Apply(_ context.Context, members []RemoteMember, now time.Time) (*Result, error) {
	s.applied = members
	s.result = &Result{Members: len(members), SyncedAt: now}
	return s.result, nil
}

type stubFetcher struct {
	members []RemoteMember
	err     error
}

func (f stubFetcher) FetchMembers(context.Context) ([]RemoteMember, error) {
	return f.members, f.err
}

func newTestService(store *stubStore, fetch Fetcher) *Service {
	svc := NewService(store, "http://contoh.invalid", "")
	svc.newFetcher = func(string, string) Fetcher { return fetch }
	return svc
}

func TestRunRequiresToken(t *testing.T) {
	svc := newTestService(&stubStore{token: ""}, stubFetcher{})

	_, err := svc.Run(t.Context())
	if err == nil {
		t.Fatal("tanpa token seharusnya gagal")
	}
	var he *httpx.Error
	if !as(err, &he) || he.Status != http.StatusServiceUnavailable {
		t.Errorf("harusnya 503, dapat %v", err)
	}
}

// An empty snapshot must NOT be applied: it would deactivate every member. An
// upstream having a bad day is far likelier than the organisation losing all
// its members at once.
func TestRunRefusesEmptySnapshot(t *testing.T) {
	store := &stubStore{token: "ada"}
	svc := newTestService(store, stubFetcher{members: nil})

	_, err := svc.Run(t.Context())
	if err == nil {
		t.Fatal("snapshot kosong seharusnya ditolak")
	}
	if store.applied != nil {
		t.Error("snapshot kosong tidak boleh sampai ke database")
	}
	var he *httpx.Error
	if !as(err, &he) || he.Status != http.StatusBadGateway {
		t.Errorf("harusnya 502, dapat %v", err)
	}
}

func TestRunAppliesSnapshot(t *testing.T) {
	store := &stubStore{token: "ada"}
	svc := newTestService(store, stubFetcher{members: []RemoteMember{
		{ID: "mem-1", ChapterID: "ch-1", Chapter: "Garuda", Name: "Budi", Status: "active"},
		{ID: "mem-2", ChapterID: "ch-1", Chapter: "Garuda", Name: "Siti", Status: "pending"},
	}})

	result, err := svc.Run(t.Context())
	if err != nil {
		t.Fatalf("sinkronisasi: %v", err)
	}
	if result.Members != 2 {
		t.Errorf("harusnya 2 member, dapat %d", result.Members)
	}
	if len(store.applied) != 2 {
		t.Errorf("snapshot tidak diteruskan ke store: %+v", store.applied)
	}
}

// The environment token is a fallback; app_settings wins so the key can be
// rotated from the settings page without a redeploy.
func TestRunPrefersSettingOverEnv(t *testing.T) {
	var usedToken string
	store := &stubStore{token: "dari-pengaturan"}
	svc := NewService(store, "http://contoh.invalid", "dari-env")
	svc.newFetcher = func(_ string, token string) Fetcher {
		usedToken = token
		return stubFetcher{members: []RemoteMember{{ID: "m", ChapterID: "c", Name: "n"}}}
	}

	if _, err := svc.Run(t.Context()); err != nil {
		t.Fatalf("sinkronisasi: %v", err)
	}
	if usedToken != "dari-pengaturan" {
		t.Errorf("harusnya memakai token dari app_settings, dapat %q", usedToken)
	}

	store.token = ""
	if _, err := svc.Run(t.Context()); err != nil {
		t.Fatalf("sinkronisasi: %v", err)
	}
	if usedToken != "dari-env" {
		t.Errorf("tanpa setting harusnya jatuh ke env, dapat %q", usedToken)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func as(err error, target **httpx.Error) bool {
	for err != nil {
		if he, ok := err.(*httpx.Error); ok {
			*target = he
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
