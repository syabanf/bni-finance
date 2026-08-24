// Package upload stores payment proofs on the local filesystem.
package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// allowedTypes is an allowlist, not a blocklist: a proof of payment is a photo
// or a PDF, and anything else has no business being written to disk here.
var allowedTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/heic":      ".heic",
	"application/pdf": ".pdf",
}

// URLPrefix is where stored files are served from.
const URLPrefix = "/uploads/"

type Store struct {
	dir     string
	maxSize int64
}

// NewStore creates the storage directory if it doesn't exist.
func NewStore(dir string, maxSize int64) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve direktori upload: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("buat direktori upload: %w", err)
	}
	return &Store{dir: abs, maxSize: maxSize}, nil
}

func (s *Store) Dir() string { return s.dir }

// Save writes one uploaded file and returns the URL path to store alongside the
// payment. The name is generated, never taken from the client: an attacker-
// chosen filename is how you get path traversal or an overwritten file.
func (s *Store) Save(r io.Reader, contentType string) (string, error) {
	ext, ok := allowedTypes[normalizeContentType(contentType)]
	if !ok {
		return "", httpx.BadRequest("tipe berkas tidak didukung — gunakan JPG, PNG, WebP, HEIC, atau PDF")
	}

	name, err := randomName(ext)
	if err != nil {
		return "", err
	}

	path := filepath.Join(s.dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("buat berkas: %w", err)
	}
	defer f.Close()

	// Cap the copy itself; a Content-Length header is a claim, not a limit.
	written, err := io.Copy(f, io.LimitReader(r, s.maxSize+1))
	if err != nil {
		os.Remove(path)
		return "", fmt.Errorf("tulis berkas: %w", err)
	}
	if written > s.maxSize {
		os.Remove(path)
		return "", httpx.BadRequest(fmt.Sprintf("berkas melebihi batas %d MB", s.maxSize>>20))
	}
	if written == 0 {
		os.Remove(path)
		return "", httpx.BadRequest("berkas kosong")
	}

	return URLPrefix + name, nil
}

// randomName is date-prefixed so the directory stays browsable, with enough
// entropy that names can't be guessed or collide.
func randomName(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("buat nama berkas: %w", err)
	}
	return time.Now().UTC().Format("20060102") + "-" + hex.EncodeToString(buf) + ext, nil
}

func normalizeContentType(ct string) string {
	parsed, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(ct))
	}
	return strings.ToLower(parsed)
}

// --- HTTP -------------------------------------------------------------------

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/uploads", auth.RequireAdmin(h.upload))
}

// RegisterFileServer exposes stored files. Proof URLs are unguessable rather
// than access-controlled, matching the public-read bucket this replaces — the
// payment page has to render them for someone who isn't signed in.
func (h *Handler) RegisterFileServer(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir(h.store.dir))
	mux.Handle("GET "+URLPrefix, http.StripPrefix(URLPrefix, noDirListing(fs)))
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.store.maxSize+1<<20)

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, httpx.BadRequest("sertakan berkas pada field 'file'"))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename)))
	}

	url, err := h.store.Save(file, contentType)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"url": url})
}

// noDirListing keeps the storage directory from being enumerated. Proof URLs
// are unguessable rather than access-controlled, so an index of them would hand
// out every stored file at once.
//
// StripPrefix turns a request for "/uploads/" into an EMPTY path, not "/", so
// checking only for a trailing slash lets the root listing straight through.
func noDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/") {
			httpx.Fail(w, httpx.NotFound("berkas tidak ditemukan"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
