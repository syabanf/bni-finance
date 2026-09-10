package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Kolom audit pada pengaturan BIAYA tidak boleh berasal dari klien.
//
// Sebelumnya `updatedBy` diteruskan apa adanya dari body permintaan. Sudah
// dibuktikan terhadap server yang berjalan: masuk sebagai Admin Nasional lalu
// mengirim {"updatedBy":"Bukan Saya"} tersimpan persis seperti itu.
//
// Yang diaudit di sini adalah NOMINAL TAGIHAN. Jejak audit yang bisa ditulis
// sendiri oleh yang diaudit bukan jejak audit — dan lebih berbahaya daripada
// kolom kosong, karena kolom kosong tidak menuduh siapa pun.
func TestUpdatedByBiayaDiambilDariTokenBukanBody(t *testing.T) {
	s := newFullServer(t)

	status, body := s.do(t, http.MethodPatch, "/api/v1/fee-settings",
		`{"renewalFee":12700000,"updatedBy":"Orang Lain — diketik klien"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}

	var fees struct {
		RenewalFee int64   `json:"renewalFee"`
		UpdatedBy  *string `json:"updatedBy"`
	}
	if err := json.Unmarshal(body, &fees); err != nil {
		t.Fatalf("urai respons: %v", err)
	}

	if fees.UpdatedBy == nil {
		t.Fatal("updatedBy kosong — seharusnya diisi dari identitas pemanggil")
	}
	if *fees.UpdatedBy == "Orang Lain — diketik klien" {
		t.Error("updatedBy diambil dari body — jejak auditnya bisa dipalsukan")
	}
	// tokenFor menandatangani admin sebagai admin@example.com.
	if *fees.UpdatedBy != "admin@example.com" {
		t.Errorf("updatedBy = %q, seharusnya email pemilik token", *fees.UpdatedBy)
	}
	// Perubahan yang sah tetap harus tersimpan.
	if fees.RenewalFee != 12_700_000 {
		t.Errorf("renewalFee = %d, perubahan yang sah ikut hilang", fees.RenewalFee)
	}
}

// Penjaga "tidak ada field yang diubah" tidak boleh menjadi mati.
//
// updatedBy sekarang SELALU diisi server. Kalau ia ikut dihitung sebagai
// "perubahan", permintaan dengan body kosong akan lolos dan menulis baris audit
// tanpa satu pun perubahan nyata.
func TestBodyKosongTetapDitolak(t *testing.T) {
	s := newFullServer(t)

	status, body := s.do(t, http.MethodPatch, "/api/v1/fee-settings", `{}`)
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, seharusnya 400 — penjaganya mati karena updatedBy selalu terisi.\nbody: %s",
			status, body)
	}
}
