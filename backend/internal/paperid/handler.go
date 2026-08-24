package paperid

import (
	"io"
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterProtected wires the admin send action and the test console onto the
// authenticated mux.
func (h *Handler) RegisterProtected(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/invoices/{id}/send", auth.RequireAdmin(h.send))
	// RequireWrite, bukan RequireAdmin: ST harus bisa mengirim invoice
	// chapternya sendiri. Batas chapternya ditegakkan di dalam Send, lewat
	// query yang membaca invoicenya — bukan di middleware ini.
	mux.HandleFunc("POST /api/v1/invoices/send-bulk", auth.RequireWrite(h.sendBulk))
	mux.HandleFunc("POST /api/v1/invoices/{id}/remind", auth.RequireAdmin(h.remind))
	mux.HandleFunc("GET /api/v1/paperid/status", auth.RequireAdmin(h.status))
	mux.HandleFunc("POST /api/v1/paperid/test-invoice", auth.RequireAdmin(h.testInvoice))
	mux.HandleFunc("POST /api/v1/paperid/test-callback", auth.RequireAdmin(h.testCallback))
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (h *Handler) testInvoice(w http.ResponseWriter, r *http.Request) {
	var in TestInvoiceInput
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &in); err != nil {
			httpx.Fail(w, err)
			return
		}
	}
	res, err := h.svc.TestInvoice(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) testCallback(w http.ResponseWriter, r *http.Request) {
	var in TestCallbackInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	res, err := h.svc.TestCallback(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// RegisterPublic wires the payment callback. Paper.id calls it with no login,
// so it must sit outside the auth middleware — it authenticates itself with the
// shared secret carried in the callback URL.
// Paper.id mendaftarkan URL callback TERPISAH per jenis kejadian di dashboard
// mereka, jadi kita menyediakan satu endpoint untuk masing-masing. Alamat yang
// berbeda membuat halaman blackbox langsung memberi tahu kejadian apa yang
// datang, tanpa perlu menebak dari isi payloadnya.
//
// Payload Payment In dan Payment Out identik — dokumentasi Paper.id menyebutnya
// eksplisit — dan hanya URL pendaftarannya yang membedakan. Itulah alasan
// keduanya perlu alamat sendiri meski penanganannya sama.
//
// Tidak ada endpoint /reconciliation: dashboard Paper.id tidak punya bagian
// untuk mendaftarkannya. Callback rekonsiliasi datang lewat bagian "Static VA
// dan Bank Email Reconciliation", dan payloadnya menandai dirinya dengan
// source "recon_static_va" — jadi /static-va yang menanganinya. Endpoint yang
// tidak bisa didaftarkan di mana pun hanyalah permukaan mati.
//
// Jenis yang tidak menyentuh penagihan kita — pembayaran keluar, pembayaran ke
// supplier, paylater, disbursement — tetap diterima dan direkam, lalu dijawab 200. Menolaknya akan
// membuat Paper.id mengulang-ulang pengiriman untuk kejadian yang memang bukan
// urusan sistem ini, dan rekamannya tetap berguna bila suatu saat dipakai.
func (h *Handler) RegisterPublic(mux *http.ServeMux) {
	// Alamat lama, dipertahankan supaya pendaftaran yang sudah ada tidak putus.
	mux.HandleFunc("POST /api/v1/webhooks/paperid", h.webhook)

	// Menyentuh invoice: bisa melunasi. Ini uang MASUK.
	mux.HandleFunc("POST /api/v1/webhooks/paperid/payment-in", h.webhook)
	mux.HandleFunc("POST /api/v1/webhooks/paperid/invoice-paid", h.webhook)
	mux.HandleFunc("POST /api/v1/webhooks/paperid/static-va", h.webhook)

	// Tidak menyentuh invoice: direkam dan diakui saja.
	//
	// payment-out termasuk di sini, dan itu KEPUTUSAN, bukan kelalaian.
	// Pembayaran keluar adalah uang yang kita bayarkan, bukan yang kita terima —
	// sistem ini hanya menagih iuran keanggotaan dan tidak pernah membayar
	// siapa pun. Dokumentasi Paper.id menyebut payload masuk dan keluar
	// IDENTIK, sehingga URL-nya adalah SATU-SATUNYA penanda arah. Merutekan
	// keduanya ke penangan yang melunasi berarti membuang penanda itu, dan satu
	// callback pembayaran keluar yang kebetulan menyebut nomor invoice kita
	// akan menandainya lunas tanpa uang pernah masuk.
	mux.HandleFunc("POST /api/v1/webhooks/paperid/payment-out", h.acknowledge)
	mux.HandleFunc("POST /api/v1/webhooks/paperid/supplier-payment", h.acknowledge)
	mux.HandleFunc("POST /api/v1/webhooks/paperid/paylater", h.acknowledge)
	mux.HandleFunc("POST /api/v1/webhooks/paperid/disbursement", h.acknowledge)
	mux.HandleFunc("POST /api/v1/webhooks/paperid/invoice-amount-due", h.acknowledge)
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	// Body is optional — default is to send no email/WhatsApp.
	var opts SendOptions
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &opts); err != nil {
			httpx.Fail(w, err)
			return
		}
	}
	inv, err := h.svc.Send(r.Context(), r.PathValue("id"), opts)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

// remind mengirim ULANG invoice yang sudah diterbitkan. Bentuk permintaannya
// sengaja sama dengan send, sehingga pemanggil tidak perlu belajar dua bentuk.
func (h *Handler) remind(w http.ResponseWriter, r *http.Request) {
	var opts SendOptions
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &opts); err != nil {
			httpx.Fail(w, err)
			return
		}
	}
	inv, err := h.svc.Remind(r.Context(), r.PathValue("id"), opts)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

// acknowledge merekam callback lalu menjawab 200 tanpa menyentuh invoice.
//
// Dipakai untuk kejadian Paper.id yang bukan urusan penagihan keanggotaan.
// Tetap diverifikasi tokennya: endpoint terbuka yang menerima apa saja adalah
// tempat menumpuknya sampah, dan rekaman yang tidak bisa dipercaya asalnya
// tidak berguna saat dipakai menelusuri masalah.
func (h *Handler) acknowledge(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Fail(w, httpx.BadRequest("gagal membaca body callback"))
		return
	}
	token := r.Header.Get("x-paper-callback-token")
	if token == "" {
		token = httpx.Query(r, "token")
	}
	if err := h.svc.AcknowledgeWebhook(r.Context(), r.URL.Path, token, raw); err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"received": true})
}

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	// Body dibaca MENTAH, bukan langsung di-decode ke struct.
	//
	// Bentuk payload Paper.id disusun dari dokumentasi dan belum pernah
	// diverifikasi terhadap callback sungguhan. Kalau berbeda, decode ke struct
	// membuang seluruh bukti: field tak dikenal hilang tanpa jejak, dan yang
	// tersisa untuk didiagnosis hanyalah struct kosong. Body mentah inilah yang
	// disimpan ke blackbox beserta catatan selisihnya.
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Fail(w, httpx.BadRequest("gagal membaca body callback"))
		return
	}

	// The secret arrives via header or ?token= — whichever the callback URL
	// registered in the Paper.id dashboard uses.
	token := r.Header.Get("x-paper-callback-token")
	if token == "" {
		token = httpx.Query(r, "token")
	}

	settled, err := h.svc.HandleWebhook(r.Context(), r.URL.Path, token, raw)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	// Always 200 on an authentic, well-formed callback — including ignored
	// events. A non-2xx just makes Paper.id retry something we chose to skip.
	httpx.JSON(w, http.StatusOK, map[string]bool{"settled": settled})
}

func (h *Handler) sendBulk(w http.ResponseWriter, r *http.Request) {
	var in BulkInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	hasil, err := h.svc.SendBulk(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	// 200 meski sebagian gagal. Kegagalan per invoice ada di dalam Baris, dan
	// menjawab 4xx/5xx akan membuat klien mengira tidak ada satu pun yang
	// terkirim — padahal nomornya sudah terbakar.
	httpx.JSON(w, http.StatusOK, hasil)
}
