package upload

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatalf("buat store: %v", err)
	}
	return store
}

func TestSaveAllowsOnlyExpectedTypes(t *testing.T) {
	store := newTestStore(t)

	ok := map[string]string{
		"image/jpeg":               ".jpg",
		"image/png":                ".png",
		"image/webp":               ".webp",
		"application/pdf":          ".pdf",
		"image/png; charset=utf-8": ".png", // parameter tambahan tetap diterima
	}
	for contentType, ext := range ok {
		url, err := store.Save(strings.NewReader("isi berkas"), contentType)
		if err != nil {
			t.Errorf("%s ditolak: %v", contentType, err)
			continue
		}
		if !strings.HasSuffix(url, ext) {
			t.Errorf("%s harusnya berakhiran %s, dapat %s", contentType, ext, url)
		}
	}

	// An allowlist, so anything executable or unknown is refused.
	for _, contentType := range []string{
		"application/x-msdownload", "text/html", "application/javascript",
		"application/octet-stream", "", "image/svg+xml",
	} {
		if _, err := store.Save(strings.NewReader("x"), contentType); err == nil {
			t.Errorf("tipe %q seharusnya ditolak", contentType)
		}
	}
}

func TestSaveEnforcesSizeLimit(t *testing.T) {
	store, err := NewStore(t.TempDir(), 64)
	if err != nil {
		t.Fatalf("buat store: %v", err)
	}

	if _, err := store.Save(bytes.NewReader(make([]byte, 65)), "image/png"); err == nil {
		t.Error("berkas melebihi batas seharusnya ditolak")
	}
	// The oversized attempt must not leave a partial file behind.
	entries, _ := os.ReadDir(store.Dir())
	if len(entries) != 0 {
		t.Errorf("berkas gagal masih tertinggal di disk: %v", entries)
	}

	if _, err := store.Save(strings.NewReader("kecil"), "image/png"); err != nil {
		t.Errorf("berkas kecil ditolak: %v", err)
	}
	if _, err := store.Save(strings.NewReader(""), "image/png"); err == nil {
		t.Error("berkas kosong seharusnya ditolak")
	}
}

// Names are generated, never taken from the client — a caller-chosen filename
// is how you get path traversal or an overwritten file.
func TestSavedNamesAreGeneratedAndUnique(t *testing.T) {
	store := newTestStore(t)

	first, err := store.Save(strings.NewReader("a"), "image/png")
	if err != nil {
		t.Fatalf("simpan: %v", err)
	}
	second, err := store.Save(strings.NewReader("b"), "image/png")
	if err != nil {
		t.Fatalf("simpan: %v", err)
	}
	if first == second {
		t.Error("dua unggahan menghasilkan nama sama")
	}
	for _, url := range []string{first, second} {
		name := strings.TrimPrefix(url, URLPrefix)
		if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
			t.Errorf("nama berkas tidak aman: %q", name)
		}
		if _, err := os.Stat(filepath.Join(store.Dir(), name)); err != nil {
			t.Errorf("berkas %q tidak ada di disk: %v", name, err)
		}
	}
}

// The stored directory must never be listable: proof URLs are unguessable
// rather than access-controlled, so an index would hand out every file at once.
func TestFileServerRefusesDirectoryListing(t *testing.T) {
	store := newTestStore(t)
	url, err := store.Save(strings.NewReader("bukti"), "image/png")
	if err != nil {
		t.Fatalf("simpan: %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store).RegisterFileServer(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	// StripPrefix turns "/uploads/" into an EMPTY path, which is why this case
	// slipped past a trailing-slash-only check once.
	for _, path := range []string{"/uploads/", "/uploads/./", "/uploads/subdir/"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body := make([]byte, 256)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Errorf("GET %s mengembalikan 200 — direktori bisa dijelajahi: %s", path, body[:n])
		}
	}

	// A known file still has to be reachable, or the payment page breaks.
	resp, err := http.Get(srv.URL + url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("berkas yang ada harus terbaca, dapat %d", resp.StatusCode)
	}
}
