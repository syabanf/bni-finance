package renewal

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/mailer"
)

// Store adalah kontrak persistensi yang dibutuhkan service.
type Store interface {
	List(ctx context.Context, f domain.RenewalFilter) ([]domain.RenewalRequest, int, error)
	GetByID(ctx context.Context, id string) (*domain.RenewalRequest, error)
	Create(ctx context.Context, memberIDs []string, period, requestedBy string, assignedMC *string) (*HasilCreate, error)
	PenerimaMC(ctx context.Context, chapterIDs []string) ([]PenerimaMC, error)
	Answer(ctx context.Context, id string, in domain.AnswerRenewalInput, answeredBy string) (*domain.RenewalRequest, error)
}

var _ Store = (*Repository)(nil)

// HasilCreate adalah hasil simpan satu permintaan massal.
type HasilCreate struct {
	Dibuat, Dilewati, Visitor int
	// BelumDijawab menghitung, per chapter, member yang permintaannya menunggu
	// jawaban MC dalam panggilan ini. Inilah yang diberitahukan lewat email.
	BelumDijawab map[string]int
}

// PenerimaMC adalah akun MC yang menerima pemberitahuan.
type PenerimaMC struct {
	ID, Email, Nama, ChapterID, NamaChapter string
}

// Surat adalah bagian mailer yang dipakai service.
type Surat interface {
	Siap() bool
	Kirim(ctx context.Context, p mailer.Pesan) error
}

// batasEmail membatasi seluruh pengiriman satu permintaan. Mailer mengulang
// tiap pesan sampai tiga kali, dan ST menunggu jawabannya.
const batasEmail = 30 * time.Second

type Service struct {
	repo    Store
	surat   Surat
	baseURL string
}

func NewService(repo Store, surat Surat, baseURL string) *Service {
	return &Service{repo: repo, surat: surat, baseURL: strings.TrimRight(baseURL, "/")}
}

func (s *Service) List(ctx context.Context, f domain.RenewalFilter) ([]domain.RenewalRequest, int, error) {
	return s.repo.List(ctx, f)
}

// HasilMinta merangkum satu permintaan massal.
type HasilMinta struct {
	Dibuat   int `json:"dibuat"`
	Dilewati int `json:"dilewati"`
	// Visitor dihitung terpisah dari Dilewati.
	//
	// "Dilewati" berarti permintaannya memang sudah ada — pekerjaan yang sudah
	// beres. Tamu yang terlewat bukan itu: ia perlu dilihat orang, karena
	// mungkin memang salah pilih, atau memang statusnya yang belum diperbarui
	// setelah ia mendaftar.
	Visitor int `json:"visitor"`
	Total   int `json:"total"`
	// Email melaporkan pemberitahuan ke MC. Permintaannya tetap tersimpan bila
	// email gagal, jadi kegagalan itu dilaporkan di sini, bukan sebagai galat.
	Email HasilEmail `json:"email"`
}

// HasilEmail menyebut apa yang terjadi pada pemberitahuan ke MC.
type HasilEmail struct {
	Terkirim           int      `json:"terkirim"`
	Gagal              int      `json:"gagal"`
	BelumDikonfigurasi bool     `json:"belumDikonfigurasi"`
	ChapterTanpaMC     []string `json:"chapterTanpaMc"`
}

// Minta membuat permintaan konfirmasi untuk sekumpulan member.
func (s *Service) Minta(ctx context.Context, in domain.CreateRenewalRequestInput) (*HasilMinta, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httpx.Unauthorized("token tidak disertakan")
	}

	period := strings.TrimSpace(in.Period)
	h, err := s.repo.Create(ctx, dedup(in.MemberIDs), period, user.ID, in.AssignedMC)
	if err != nil {
		return nil, err
	}
	return &HasilMinta{
		Dibuat: h.Dibuat, Dilewati: h.Dilewati, Visitor: h.Visitor,
		Total: h.Dibuat + h.Dilewati + h.Visitor,
		Email: s.beritahuMC(ctx, h.BelumDijawab, period, in.AssignedMC),
	}, nil
}

// beritahuMC mengirim satu email per MC, berisi jumlah member di chapternya.
//
// Tiap chapter diberi tahu ke MC-nya sendiri. MC yang ditugaskan menjadi
// satu-satunya penerima di chapternya; chapter lain dalam permintaan yang sama
// tetap diberi tahu ke seluruh MC-nya.
func (s *Service) beritahuMC(ctx context.Context, belumDijawab map[string]int, period string, assignedMC *string) HasilEmail {
	hasil := HasilEmail{ChapterTanpaMC: []string{}}
	if len(belumDijawab) == 0 {
		return hasil
	}
	ids := make([]string, 0, len(belumDijawab))
	for id := range belumDijawab {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	semua, err := s.repo.PenerimaMC(ctx, ids)
	if err != nil {
		slog.Error("akun MC tidak terbaca, email konfirmasi renewal tidak dikirim", "error", err)
		hasil.Gagal = len(ids)
		return hasil
	}
	penerima := saringPenerima(semua, assignedMC)

	for _, id := range ids {
		if !slices.ContainsFunc(penerima, func(p PenerimaMC) bool { return p.ChapterID == id }) {
			hasil.ChapterTanpaMC = append(hasil.ChapterTanpaMC, id)
		}
	}
	if !s.surat.Siap() {
		hasil.BelumDikonfigurasi = true
		return hasil
	}

	// Permintaannya sudah tersimpan, jadi ST yang menutup halaman tidak boleh
	// membatalkan email yang tersisa.
	ctxKirim, batal := context.WithTimeout(context.WithoutCancel(ctx), batasEmail)
	defer batal()
	for _, p := range penerima {
		if err := s.surat.Kirim(ctxKirim, s.pesanMC(p, belumDijawab[p.ChapterID], period)); err != nil {
			slog.Warn("email konfirmasi renewal gagal", "chapter", p.ChapterID, "error", err)
			hasil.Gagal++
			continue
		}
		hasil.Terkirim++
	}
	return hasil
}

// saringPenerima menyisakan MC yang ditugaskan di chapternya. MC yang tidak ada
// di daftar diabaikan, supaya chapter itu tetap diberi tahu ke seluruh MC-nya.
func saringPenerima(semua []PenerimaMC, assignedMC *string) []PenerimaMC {
	if assignedMC == nil {
		return semua
	}
	i := slices.IndexFunc(semua, func(p PenerimaMC) bool { return p.ID == *assignedMC })
	if i < 0 {
		return semua
	}
	chapter := semua[i].ChapterID
	return slices.DeleteFunc(slices.Clone(semua), func(p PenerimaMC) bool {
		return p.ChapterID == chapter && p.ID != *assignedMC
	})
}

// labelPeriode menulis "2027-S2" sebagai "Jul–Des 2027". Periode lama yang
// berupa tahun saja dibiarkan apa adanya.
func labelPeriode(p string) string {
	switch {
	case strings.HasSuffix(p, "-S1"):
		return "Jan–Jun " + strings.TrimSuffix(p, "-S1")
	case strings.HasSuffix(p, "-S2"):
		return "Jul–Des " + strings.TrimSuffix(p, "-S2")
	}
	return p
}

func (s *Service) pesanMC(p PenerimaMC, jumlah int, period string) mailer.Pesan {
	label := labelPeriode(period)
	tautan := s.baseURL + "/renewal?periode=" + url.QueryEscape(period)
	ringkas := fmt.Sprintf("%d member %s menunggu jawaban Anda untuk periode %s.", jumlah, p.NamaChapter, label)
	return mailer.Pesan{
		Ke:     p.Email,
		Jenis:  "konfirmasi-renewal",
		Subjek: fmt.Sprintf("Konfirmasi renewal %s: %d member menunggu jawaban", label, jumlah),
		Teks: "Halo " + p.Nama + ",\r\n\r\n" + ringkas + "\r\n\r\n" +
			"Buka halaman Konfirmasi Renewal untuk menjawab:\r\n" + tautan + "\r\n",
		HTML: `<div style="font:15px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#1a1a1a">` +
			`<p>Halo ` + html.EscapeString(p.Nama) + `,</p>` +
			`<p>` + html.EscapeString(ringkas) + `</p>` +
			`<p><a href="` + html.EscapeString(tautan) + `" style="color:#c8102e">Buka Konfirmasi Renewal</a></p></div>`,
	}
}

// Jawab mencatat jawaban MC.
//
// SIAPA yang boleh menjawab dijaga di sini, bukan di middleware: middleware
// hanya tahu peran, sedangkan aturan ini juga menyangkut keadaan barisnya.
func (s *Service) Jawab(ctx context.Context, id string, in domain.AnswerRenewalInput) (*domain.RenewalRequest, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httpx.Unauthorized("token tidak disertakan")
	}
	// MC dan admin yang menjawab. ST TIDAK, dan itu inti alurnya: ST yang
	// menanyakan, jadi membiarkannya menjawab sendiri membuat konfirmasinya
	// tidak berarti apa-apa — ia hanya mencatat dugaannya sendiri sebagai
	// jawaban orang lain.
	if user.Role != domain.RoleMC && user.Role != domain.RoleAdmin {
		return nil, httpx.Forbidden("hanya MC yang menjawab konfirmasi renewal")
	}
	return s.repo.Answer(ctx, id, in, user.ID)
}

func dedup(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// --- HTTP -------------------------------------------------------------------

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/renewal-requests", h.list)
	// RequireWrite: admin dan ST. MC tidak boleh MEMINTA konfirmasi kepada
	// dirinya sendiri.
	mux.HandleFunc("POST /api/v1/renewal-requests", auth.RequireWrite(h.minta))
	// Menjawab TIDAK memakai RequireWrite: MC hanya boleh membaca di tempat
	// lain, tapi harus boleh menjawab di sini. Perannya diperiksa di service.
	mux.HandleFunc("PATCH /api/v1/renewal-requests/{id}", h.jawab)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.RenewalFilter{
		ChapterID: httpx.Query(r, "chapterId"),
		Answer:    httpx.Query(r, "answer"),
		Period:    httpx.Query(r, "period"),
		Limit:     httpx.QueryInt(r, "limit", 50, 1, 200),
		Offset:    httpx.QueryInt(r, "offset", 0, 0, 1_000_000),
	}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, f.Limit, f.Offset))
}

func (h *Handler) minta(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateRenewalRequestInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	out, err := h.svc.Minta(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, out)
}

func (h *Handler) jawab(w http.ResponseWriter, r *http.Request) {
	var in domain.AnswerRenewalInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	out, err := h.svc.Jawab(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
