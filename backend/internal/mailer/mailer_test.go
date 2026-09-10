package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// resendPalsu menjalankan API tiruan dan mengembalikan mailer yang menunjuk ke
// sana, badan permintaan yang diterima, dan pencacah percobaan.
func resendPalsu(t *testing.T, tangani func(w http.ResponseWriter, r *http.Request)) (*Mailer, *[]byte, *atomic.Int32) {
	t.Helper()
	var diterima []byte
	var cacah atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cacah.Add(1)
		b, _ := io.ReadAll(r.Body)
		diterima = b
		if tangani != nil {
			tangani(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pesan-1"}`))
	}))
	t.Cleanup(srv.Close)

	m := New(Config{APIKey: "re_tes", From: "Pengirim <kirim@contoh.test>", BaseURL: srv.URL})
	m.jeda = 0 // tanpa jeda: yang diuji jumlah percobaannya, bukan durasinya
	return m, &diterima, &cacah
}

func badan(t *testing.T, mentah []byte) badanKirim {
	t.Helper()
	var b badanKirim
	if err := json.Unmarshal(mentah, &b); err != nil {
		t.Fatalf("badan bukan JSON: %v\n%s", err, mentah)
	}
	return b
}

// HEADER TIDAK BOLEH BISA DISUNTIK.
//
// Subjek dan alamat penerima berasal dari data pengguna. Resend yang merakit
// pesan RFC 5322 di sisinya, jadi baris baru yang lolos dari sini adalah header
// tambahan yang KITA titipkan — sebuah Bcc, misalnya, yang mengirim salinan
// setiap tautan reset kata sandi ke alamat penyerang.
//
// Dijaga di sini alih-alih dipercayakan ke perakit di seberang: perilakunya
// bukan milik kita, tidak terlihat dari repositori ini, dan bisa berubah tanpa
// kita tahu.
func TestHeaderTidakBisaDisuntik(t *testing.T) {
	m, mentah, _ := resendPalsu(t, nil)

	err := m.Kirim(context.Background(), Pesan{
		Ke:     "korban@contoh.test\r\nBcc: penyerang@jahat.test",
		Subjek: "Reset\r\nBcc: penyerang@jahat.test",
		Teks:   "isi",
	})
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}

	b := badan(t, *mentah)
	for _, v := range append([]string{b.From, b.Subject}, b.To...) {
		if strings.ContainsAny(v, "\r\n") {
			t.Errorf("baris baru lolos ke header %q", v)
		}
	}
	// Isinya tidak boleh hilang — hanya baris barunya yang diratakan jadi
	// spasi, supaya jejak percobaannya masih terbaca oleh yang memeriksa.
	if b.Subject != "Reset Bcc: penyerang@jahat.test" {
		t.Errorf("subjek = %q; baris barunya seharusnya jadi satu spasi, bukan dibuang", b.Subject)
	}
	if len(b.To) != 1 {
		t.Errorf("penerima = %v, seharusnya tepat satu", b.To)
	}
}

// HTML selalu disertai versi teks.
//
// Klien yang menolak HTML — dan penyaring spam yang mencurigai HTML tanpa
// alternatif — akan menampilkan bagian teksnya. Email verifikasi yang berakhir
// di folder spam sama saja dengan tidak terkirim.
func TestHTMLSelaluDisertaiTeks(t *testing.T) {
	m, mentah, _ := resendPalsu(t, nil)
	err := m.Kirim(context.Background(), Pesan{
		Ke: "a@b.test", Subjek: "s", Teks: "versi teks", HTML: "<p>versi html</p>",
	})
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}
	b := badan(t, *mentah)
	if b.Text != "versi teks" || b.HTML != "<p>versi html</p>" {
		t.Errorf("text=%q html=%q — keduanya harus terkirim", b.Text, b.HTML)
	}
}

// Pesan teks-saja tidak boleh membawa field html kosong: Resend memperlakukan
// html:"" sebagai bagian HTML yang ada tapi kosong, dan sebagian klien
// menampilkan email kosong alih-alih teksnya.
func TestTeksSajaTidakMengirimHTMLKosong(t *testing.T) {
	m, mentah, _ := resendPalsu(t, nil)
	if err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "halo"}); err != nil {
		t.Fatalf("kirim: %v", err)
	}
	var peta map[string]any
	if err := json.Unmarshal(*mentah, &peta); err != nil {
		t.Fatalf("badan bukan JSON: %v", err)
	}
	if _, ada := peta["html"]; ada {
		t.Errorf("field html ikut terkirim padahal pesannya teks-saja: %s", *mentah)
	}
}

// Kunci API harus berangkat sebagai Bearer. Tanpa ini permintaannya ditolak
// 401, dan tesnya yang lain tidak akan menangkapnya karena servernya tiruan.
func TestKunciAPIDikirimSebagaiBearer(t *testing.T) {
	var punya string
	m, _, _ := resendPalsu(t, func(w http.ResponseWriter, r *http.Request) {
		punya = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"id":"x"}`))
	})
	if err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"}); err != nil {
		t.Fatalf("kirim: %v", err)
	}
	if punya != "Bearer re_tes" {
		t.Errorf("Authorization = %q, seharusnya %q", punya, "Bearer re_tes")
	}
}

// Konfigurasi setengah jadi harus ditolak SEBELUM menyentuh jaringan, dengan
// galat yang bisa dibedakan dari kegagalan kirim.
func TestBelumDikonfigurasiDitolakJelas(t *testing.T) {
	kasus := []struct {
		nama string
		cfg  Config
	}{
		{"kunci kosong", Config{From: "a@b.test"}},
		// From tanpa alamat lolos dari pemeriksaan "tidak kosong" tapi pasti
		// ditolak Resend — dan ditolaknya jauh dari tempat orang melihatnya.
		{"From tanpa alamat", Config{APIKey: "re_x", From: "BNI Finance Hub"}},
		{"From kosong", Config{APIKey: "re_x"}},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			// BaseURL sengaja menunjuk alamat mati: kalau penjaganya bocor,
			// tesnya gagal karena galat jaringan, bukan lewat diam-diam.
			k.cfg.BaseURL = "http://127.0.0.1:1"
			m := New(k.cfg)
			if m.Siap() {
				t.Fatal("Siap() true padahal konfigurasinya belum lengkap")
			}
			err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"})
			if err == nil {
				t.Fatal("kirim berhasil padahal email belum dikonfigurasi")
			}
			if !strings.Contains(err.Error(), "belum dikonfigurasi") {
				t.Errorf("galat = %v, seharusnya menyebut belum dikonfigurasi", err)
			}
		})
	}
}

// PENJELASAN RESEND HARUS SAMPAI KE PEMANGGIL.
//
// Inilah alasan pindah dari SMTP. Domain yang belum diverifikasi, kunci yang
// dicabut, alamat yang salah bentuk — semuanya punya pesan yang menyebutkan
// perbaikannya, dan membuangnya jadi "gagal kirim" mengubah masalah lima menit
// jadi masalah setengah hari.
func TestGalatResendDisampaikanApaAdanya(t *testing.T) {
	m, _, _ := resendPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"statusCode":403,"name":"validation_error",` +
			`"message":"The reddie.id domain is not verified. Please verify your domain."}`))
	})
	err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"})
	if err == nil {
		t.Fatal("kirim berhasil padahal dijawab 403")
	}
	// Bentuknya diperiksa, bukan sekadar potongan katanya.
	//
	// Asersi pertama saya mencari "403", "validation_error", dan "domain is not
	// verified" secara terpisah — dan LULUS walau penjelasan Resend dibuang,
	// karena badan JSON mentahnya memuat ketiganya juga. Tes yang lulus pada
	// perilaku yang salah tidak menjaga apa pun.
	mau := "403 validation_error: The reddie.id domain is not verified. Please verify your domain."
	if !strings.Contains(err.Error(), mau) {
		t.Errorf("galat = %q\nseharusnya memuat %q", err, mau)
	}
	if strings.Contains(err.Error(), `"statusCode"`) {
		t.Errorf("badan JSON mentah ikut dibuang ke pesan galat: %q", err)
	}
}

// 2xx tanpa id BUKAN keberhasilan.
//
// Tanpa id, tidak ada yang bisa ditelusuri di dasbor Resend — dan melaporkannya
// sebagai berhasil berarti mengarang bukti pengiriman untuk email yang mungkin
// tidak pernah ada. Perantara (proxy perusahaan, halaman penyedia yang sedang
// gangguan) memang bisa menjawab 200 dengan badan yang bukan dari Resend.
func TestJawabanSuksesTanpaIDDitolak(t *testing.T) {
	m, _, _ := resendPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"})
	if err == nil {
		t.Fatal("2xx tanpa id dilaporkan berhasil")
	}
	if !strings.Contains(err.Error(), "tanpa id") {
		t.Errorf("galat = %v, seharusnya menyebut id yang hilang", err)
	}
}

// 429 diulang; batas Resend 2 permintaan/detik dan pengiriman massal
// menabraknya dengan mudah. Menyerah pada percobaan pertama berarti sebagian
// penerima dalam satu blast tidak pernah menerima apa-apa.
func TestBatasLajuDiulang(t *testing.T) {
	var n atomic.Int32
	m, _, cacah := resendPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"statusCode":429,"message":"Too many requests"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"pesan-2"}`))
	})

	id, err := m.KirimDenganID(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"})
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}
	if id != "pesan-2" {
		t.Errorf("id = %q, seharusnya dari percobaan kedua", id)
	}
	if got := cacah.Load(); got != 2 {
		t.Errorf("percobaan = %d, seharusnya 2", got)
	}
}

// 5xx TIDAK diulang.
//
// Ia ambigu: pesannya mungkin sudah masuk antrean di sana dan hanya jawabannya
// yang gagal. Mengulanginya bisa mengirim tautan reset KEDUA yang sama-sama
// sah ke orang yang sama — dua kunci beredar lebih buruk daripada satu email
// gagal yang bisa diminta ulang oleh orangnya sendiri.
func TestGalatServerTidakDiulang(t *testing.T) {
	m, _, cacah := resendPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"statusCode":500,"message":"Internal error"}`))
	})
	if err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"}); err == nil {
		t.Fatal("500 dilaporkan berhasil")
	}
	if got := cacah.Load(); got != 1 {
		t.Errorf("percobaan = %d, seharusnya 1 — 5xx ambigu, mengulangnya bisa menggandakan email", got)
	}
}

// Penerima kosong ditolak sebelum jaringan: permintaan tanpa tujuan hanya
// menghabiskan kuota dan kembali sebagai galat yang membingungkan.
func TestPenerimaKosongDitolak(t *testing.T) {
	m, _, cacah := resendPalsu(t, nil)
	if err := m.Kirim(context.Background(), Pesan{Ke: "  \r\n ", Subjek: "s", Teks: "t"}); err == nil {
		t.Fatal("penerima kosong diterima")
	}
	if got := cacah.Load(); got != 0 {
		t.Errorf("percobaan = %d, seharusnya tidak menyentuh jaringan sama sekali", got)
	}
}

// SUBJEK TIDAK BOLEH SAMPAI KE LOG.
//
// Subjek email OTP berbunyi "Kode masuk 817810 — BNI Finance Hub": kodenya ada
// di dalamnya supaya terbaca dari pratinjau notifikasi tanpa membuka email.
// Mencatat subjek berarti setiap kode masuk yang masih berlaku tersimpan apa
// adanya di log aplikasi.
//
// Ini bukan bahaya yang dibawa satu perubahan. Menaruh kode di subjek masuk
// akal. Mencatat subjek — alih-alih alamat penerima — juga masuk akal, justru
// karena hati-hati soal data pribadi. Keduanya digabung tanpa satu pun konflik
// teks, dan kebocorannya baru terlihat saat log-nya dibaca.
func TestSubjekTidakMasukLog(t *testing.T) {
	m, _, _ := resendPalsu(t, nil)

	var catatan bytes.Buffer
	m.log = slog.New(slog.NewJSONHandler(&catatan, nil))

	const kode = "817810"
	err := m.Kirim(context.Background(), Pesan{
		Ke:     "a@b.test",
		Jenis:  "otp",
		Subjek: "Kode masuk " + kode + " — BNI Finance Hub",
		Teks:   "Kode masuk Anda: " + kode,
	})
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}

	log := catatan.String()
	if log == "" {
		t.Fatal("tidak ada yang tercatat — penjaga ini akan hijau selamanya")
	}
	if strings.Contains(log, kode) {
		t.Errorf("kode OTP tercatat di log:\n%s", log)
	}
	if strings.Contains(log, "Kode masuk") {
		t.Errorf("subjek tercatat di log:\n%s", log)
	}
	// Isi pesannya juga tidak, dan alamat penerimanya juga tidak.
	if strings.Contains(log, "a@b.test") {
		t.Errorf("alamat penerima tercatat di log:\n%s", log)
	}
	// Yang HARUS ada: id, supaya pesannya bisa ditelusuri di dasbor Resend,
	// dan jenisnya, supaya log-nya masih bisa dibaca orang.
	for _, harus := range []string{"pesan-1", "otp"} {
		if !strings.Contains(log, harus) {
			t.Errorf("log kehilangan %q — tanpa itu ia tidak berguna:\n%s", harus, log)
		}
	}
}
