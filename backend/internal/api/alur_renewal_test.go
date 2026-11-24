package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// Alur renewal dari klien lewat HTTP di atas Postgres: ST meminta invoice,
// finance menerbitkan, ST mengirim bukti bayar, finance memverifikasi, lalu
// finance mengajukan renewal ke BNI Global. Langkah 1 (email BNI Global) dan
// langkah 2 (konfirmasi MC) diuji di tempat lain.
//
// Setiap langkah juga memeriksa tahap yang tampil di Konfirmasi Renewal.
func TestAlurRenewalKlien(t *testing.T) {
	s := newLiveServer(t)

	s.req(t, http.MethodPost, "/api/v1/chapters",
		`{"id":"ch-alur","name":"Alur","displayName":"BNI Alur","cityName":"Jakarta"}`, http.StatusCreated)
	s.req(t, http.MethodPost, "/api/v1/chapters",
		`{"id":"ch-lain","name":"Lain","displayName":"BNI Lain","cityName":"Bandung"}`, http.StatusCreated)
	var m domain.Member
	mustJSON(t, s.req(t, http.MethodPost, "/api/v1/members",
		`{"chapterId":"ch-alur","name":"Anggota Alur","renewalDate":"`+inDays(20)+`"}`,
		http.StatusCreated), &m)
	konfirmasiRenewal(t, s.pool, m.ID)

	st := s.tokenST(t, "ch-alur")
	stLain := s.tokenST(t, "ch-lain")

	permintaan := func() domain.RenewalRequest {
		t.Helper()
		var out struct{ Data []domain.RenewalRequest }
		mustJSON(t, s.req(t, http.MethodGet, "/api/v1/renewal-requests?chapterId=ch-alur", "", http.StatusOK), &out)
		if len(out.Data) != 1 {
			t.Fatalf("permintaan konfirmasi: dapat %d baris, mau 1", len(out.Data))
		}
		return out.Data[0]
	}
	tahap := func(mau domain.RenewalStage) {
		t.Helper()
		if dapat := permintaan().Stage; dapat != mau {
			t.Fatalf("tahap: dapat %q, mau %q", dapat, mau)
		}
	}
	tahap(domain.RenewalStage(domain.RenewalWillRenew))
	idKonfirmasi := permintaan().ID
	mintaInvoice := `{"ids":["` + idKonfirmasi + `"]}`

	// Langkah 3. ST chapter lain tidak menandai apa pun, ST chapternya sendiri
	// menandai satu, dan permintaan kedua dilewati.
	var minta struct{ Diminta, Dilewati int }
	mustJSON(t, s.reqAs(t, stLain, http.MethodPost, "/api/v1/renewal-requests/invoice-requests",
		mintaInvoice, http.StatusOK), &minta)
	if minta.Diminta != 0 {
		t.Fatalf("ST chapter lain menandai %d permintaan", minta.Diminta)
	}
	mustJSON(t, s.reqAs(t, st, http.MethodPost, "/api/v1/renewal-requests/invoice-requests",
		mintaInvoice, http.StatusOK), &minta)
	if minta.Diminta != 1 {
		t.Fatalf("permintaan invoice: diminta %d, mau 1", minta.Diminta)
	}
	mustJSON(t, s.reqAs(t, st, http.MethodPost, "/api/v1/renewal-requests/invoice-requests",
		mintaInvoice, http.StatusOK), &minta)
	if minta.Diminta != 0 || minta.Dilewati != 1 {
		t.Fatalf("permintaan ulang: %+v, mau dilewati", minta)
	}
	tahap(domain.StageInvoiceRequested)
	var antrean struct{ Meta struct{ Total int } }
	mustJSON(t, s.req(t, http.MethodGet, "/api/v1/renewal-requests?stage=invoice_requested", "", http.StatusOK), &antrean)
	if antrean.Meta.Total != 1 {
		t.Fatalf("antrean invoice diminta: %d, mau 1", antrean.Meta.Total)
	}
	s.req(t, http.MethodGet, "/api/v1/renewal-requests?stage=ngawur", "", http.StatusBadRequest)

	// Menarik lalu meminta lagi.
	s.reqAs(t, st, http.MethodDelete, "/api/v1/renewal-requests/"+idKonfirmasi+"/invoice-request", "", http.StatusOK)
	tahap(domain.RenewalStage(domain.RenewalWillRenew))
	s.reqAs(t, st, http.MethodPost, "/api/v1/renewal-requests/invoice-requests", mintaInvoice, http.StatusOK)

	// Langkah 4. ST tidak menerbitkan dan tidak mengirim invoice.
	badan := `{"memberId":"` + m.ID + `","chapterId":"ch-alur","type":"renewal","amount":11523200,` +
		`"dueDate":"` + inDays(30) + `","periodStart":"` + inDays(0) + `","periodEnd":"` + inDays(365) + `"}`
	s.reqAs(t, st, http.MethodPost, "/api/v1/invoices", badan, http.StatusForbidden)
	var inv domain.Invoice
	mustJSON(t, s.req(t, http.MethodPost, "/api/v1/invoices", badan, http.StatusCreated), &inv)
	tahap(domain.StageInvoiceDraft)
	s.reqAs(t, st, http.MethodDelete, "/api/v1/renewal-requests/"+idKonfirmasi+"/invoice-request", "", http.StatusConflict)
	s.req(t, http.MethodPatch, "/api/v1/invoices/"+inv.ID, `{"status":"sent"}`, http.StatusOK)
	tahap(domain.StageInvoiced)

	// Langkah 5. ST mengirim bukti, dan tidak bisa menandai lunas sendiri.
	bukti := `{"invoiceId":"` + inv.ID + `","amount":11523200,"paidAt":"` + inDays(0) +
		`","paymentMethod":"bank_transfer","proofUrl":"/uploads/20261124-bukti.jpg"}`
	s.reqAs(t, st, http.MethodPost, "/api/v1/payments", `{"invoiceId":"`+inv.ID+`","amount":1}`, http.StatusForbidden)
	s.reqAs(t, stLain, http.MethodPost, "/api/v1/payment-proofs", bukti, http.StatusNotFound)
	s.reqAs(t, st, http.MethodPost, "/api/v1/payment-proofs",
		`{"invoiceId":"`+inv.ID+`","amount":1,"paidAt":"`+inDays(0)+`","proofUrl":"https://luar.invalid/a.jpg"}`,
		http.StatusBadRequest)
	var p1 domain.PaymentProof
	mustJSON(t, s.reqAs(t, st, http.MethodPost, "/api/v1/payment-proofs", bukti, http.StatusCreated), &p1)
	s.reqAs(t, st, http.MethodPost, "/api/v1/payment-proofs", bukti, http.StatusConflict)
	tahap(domain.StageProofPending)

	// ST tidak memverifikasi. Finance yang menolak tanpa alasan mendapat 400.
	// Dengan alasan, bukti ditolak dan invoice tetap belum lunas.
	s.reqAs(t, st, http.MethodPost, "/api/v1/payment-proofs/"+p1.ID+"/approve", `{}`, http.StatusForbidden)
	s.req(t, http.MethodPost, "/api/v1/payment-proofs/"+p1.ID+"/reject", `{}`, http.StatusBadRequest)
	var ditolak domain.PaymentProof
	mustJSON(t, s.req(t, http.MethodPost, "/api/v1/payment-proofs/"+p1.ID+"/reject",
		`{"note":"nominal tidak terbaca"}`, http.StatusOK), &ditolak)
	if ditolak.Status != domain.ProofRejected || ditolak.ReviewNote == nil {
		t.Fatalf("bukti ditolak: %+v", ditolak)
	}
	s.req(t, http.MethodPost, "/api/v1/payment-proofs/"+p1.ID+"/approve", `{}`, http.StatusConflict)
	tahap(domain.StageInvoiced)

	// Bukti kedua bisa ditarik ST selama belum diperiksa, lalu dikirim lagi.
	var p2 domain.PaymentProof
	mustJSON(t, s.reqAs(t, st, http.MethodPost, "/api/v1/payment-proofs", bukti, http.StatusCreated), &p2)
	s.reqAs(t, stLain, http.MethodDelete, "/api/v1/payment-proofs/"+p2.ID, "", http.StatusNotFound)
	s.reqAs(t, st, http.MethodDelete, "/api/v1/payment-proofs/"+p2.ID, "", http.StatusNoContent)
	var p3 domain.PaymentProof
	mustJSON(t, s.reqAs(t, st, http.MethodPost, "/api/v1/payment-proofs", bukti, http.StatusCreated), &p3)

	// Saringan dan batas chapter pada daftar bukti: ST chapter lain tidak
	// melihat apa pun, dan tiap saringan mengembalikan himpunan yang tepat.
	jumlahBukti := func(token, q string) int {
		t.Helper()
		var out struct{ Meta struct{ Total int } }
		mustJSON(t, s.reqAs(t, token, http.MethodGet, "/api/v1/payment-proofs"+q, "", http.StatusOK), &out)
		return out.Meta.Total
	}
	for _, k := range []struct {
		nama, token, q string
		mau            int
	}{
		{"ST chapter lain", stLain, "", 0},
		{"ST chapter lain, saring chapter orang", stLain, "?chapterId=ch-alur", 0},
		{"ST sendiri", st, "", 2},
		{"menunggu", st, "?status=pending", 1},
		{"ditolak", st, "?status=rejected", 1},
		{"disetujui", st, "?status=approved", 0},
		{"per invoice", s.token, "?invoiceId=" + inv.ID, 2},
		{"chapter lain oleh admin", s.token, "?chapterId=ch-lain", 0},
	} {
		if dapat := jumlahBukti(k.token, k.q); dapat != k.mau {
			t.Errorf("daftar bukti %s: dapat %d, mau %d", k.nama, dapat, k.mau)
		}
	}
	s.req(t, http.MethodGet, "/api/v1/payment-proofs?status=ngawur", "", http.StatusBadRequest)
	s.reqAs(t, stLain, http.MethodGet, "/api/v1/payment-proofs/"+p3.ID, "", http.StatusNotFound)

	var disetujui domain.PaymentProof
	mustJSON(t, s.req(t, http.MethodPost, "/api/v1/payment-proofs/"+p3.ID+"/approve", `{}`, http.StatusOK), &disetujui)
	if disetujui.Status != domain.ProofApproved || disetujui.PaymentID == nil || disetujui.InvoiceStatus != domain.StatusPaid {
		t.Fatalf("bukti disetujui: %+v", disetujui)
	}
	s.reqAs(t, st, http.MethodDelete, "/api/v1/payment-proofs/"+p3.ID, "", http.StatusConflict)
	var bayar struct{ Data []domain.Payment }
	mustJSON(t, s.req(t, http.MethodGet, "/api/v1/payments?invoiceId="+inv.ID, "", http.StatusOK), &bayar)
	if len(bayar.Data) != 1 || bayar.Data[0].ProofURL == nil {
		t.Fatalf("pembayaran dari bukti: %+v", bayar.Data)
	}
	tahap(domain.StagePaid)

	// Langkah 6. ST tidak mengajukan. Finance menandai, dan tanda kedua dilewati.
	var belum struct{ Meta struct{ Total int } }
	mustJSON(t, s.req(t, http.MethodGet, "/api/v1/invoices?submission=pending", "", http.StatusOK), &belum)
	if belum.Meta.Total != 1 {
		t.Fatalf("lunas belum diajukan: %d, mau 1", belum.Meta.Total)
	}
	ajukan := `{"ids":["` + inv.ID + `"]}`
	s.reqAs(t, st, http.MethodPost, "/api/v1/invoices/renewal-submissions", ajukan, http.StatusForbidden)
	s.req(t, http.MethodPost, "/api/v1/invoices/renewal-submissions",
		`{"ids":["`+inv.ID+`"],"submittedAt":"`+inDays(2)+`"}`, http.StatusBadRequest)
	var tanda struct{ Ditandai, Dilewati int }
	mustJSON(t, s.req(t, http.MethodPost, "/api/v1/invoices/renewal-submissions", ajukan, http.StatusOK), &tanda)
	if tanda.Ditandai != 1 {
		t.Fatalf("pengajuan: %+v", tanda)
	}
	mustJSON(t, s.req(t, http.MethodPost, "/api/v1/invoices/renewal-submissions", ajukan, http.StatusOK), &tanda)
	if tanda.Ditandai != 0 || tanda.Dilewati != 1 {
		t.Fatalf("pengajuan ulang: %+v", tanda)
	}
	tahap(domain.StageSubmitted)

	// Batas chapter pada daftar konfirmasi dan saringan pengajuan.
	for _, k := range []struct {
		nama, token, path string
		mau               int
	}{
		{"konfirmasi, ST chapter lain", stLain, "/api/v1/renewal-requests", 0},
		{"konfirmasi, ST sendiri", st, "/api/v1/renewal-requests", 1},
		{"tahap diajukan", st, "/api/v1/renewal-requests?stage=submitted", 1},
		{"tahap lunas", st, "/api/v1/renewal-requests?stage=paid", 0},
		{"sudah diajukan, ST chapter lain", stLain, "/api/v1/invoices?submission=done", 0},
		{"sudah diajukan, ST sendiri", st, "/api/v1/invoices?submission=done", 1},
		{"belum diajukan", s.token, "/api/v1/invoices?submission=pending", 0},
	} {
		var out struct{ Meta struct{ Total int } }
		mustJSON(t, s.reqAs(t, k.token, http.MethodGet, k.path, "", http.StatusOK), &out)
		if out.Meta.Total != k.mau {
			t.Errorf("%s: dapat %d, mau %d", k.nama, out.Meta.Total, k.mau)
		}
	}

	var jejak struct{ Data []domain.AuditEntry }
	mustJSON(t, s.req(t, http.MethodGet, "/api/v1/invoices/"+inv.ID+"/audit", "", http.StatusOK), &jejak)
	adaPengajuan := false
	for _, e := range jejak.Data {
		adaPengajuan = adaPengajuan || e.Action == domain.AuditRenewalSubmitted
	}
	if !adaPengajuan {
		t.Fatalf("jejak audit tanpa pengajuan renewal: %+v", jejak.Data)
	}

	// Tanda yang salah pasang bisa dibatalkan, dan pembatalan kedua ditolak.
	s.req(t, http.MethodDelete, "/api/v1/invoices/"+inv.ID+"/renewal-submission", "", http.StatusOK)
	s.req(t, http.MethodDelete, "/api/v1/invoices/"+inv.ID+"/renewal-submission", "", http.StatusConflict)
	tahap(domain.StagePaid)
}

// tokenST menandatangani token ST yang terikat ke satu chapter.
func (s *liveStack) tokenST(t *testing.T, chapterID string) string {
	t.Helper()
	token, _, err := s.signer.Sign(domain.User{
		ID: "33333333-3333-3333-3333-333333333333", Email: "st-" + chapterID + "@example.com",
		Name: "ST " + chapterID, Role: domain.RoleST, ChapterID: &chapterID,
	}, time.Now())
	if err != nil {
		t.Fatalf("tanda tangani token ST: %v", err)
	}
	return token
}

func mustJSON(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
}
