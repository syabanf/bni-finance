// Package mailer mengirim email transaksional lewat REST API Resend.
//
// Hanya net/http dari pustaka standar — tidak ada dependensi baru. Yang
// dibutuhkan aplikasi ini sempit: beberapa pesan pendek, satu penerima, tanpa
// lampiran. SDK resmi membawa retry, batch, dan tipe untuk audience/broadcast
// yang tidak satu pun dipakai di sini.
//
// # Kenapa REST, bukan SMTP
//
// Versi sebelumnya memakai net/smtp. Ia bekerja, tapi menyembunyikan kegagalan:
// percakapan SMTP membalas dengan kode tiga digit yang net/smtp bungkus jadi
// satu string, dan galat yang paling penting justru muncul PALING AWAL — saat
// MAIL FROM — di mana pesannya cuma "501 Bad sender address syntax" tanpa
// menyebut alamat mana yang ditolak.
//
// REST menjawab dengan JSON yang menyebutkan namanya sendiri:
//
//	{"statusCode":403,"name":"validation_error",
//	 "message":"The reddie.id domain is not verified..."}
//
// dan pada keberhasilan mengembalikan id pesan yang bisa dicari di dasbor
// Resend. Untuk pertanyaan yang sebenarnya diajukan orang — "email saya belum
// sampai, kenapa" — selisih itu adalah selisih antara punya jawaban dan tidak.
package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ErrTidakDikonfigurasi dikembalikan saat kredensial email belum diisi.
//
// Dibedakan dari galat pengiriman supaya pemanggilnya bisa menjawab 503 dengan
// pesan yang jelas — "belum dikonfigurasi" adalah keadaan yang bisa diperbaiki
// orang, sedangkan "gagal kirim" belum tentu.
var ErrTidakDikonfigurasi = errors.New("email belum dikonfigurasi")

const (
	baseBawaan = "https://api.resend.com"
	batasWaktu = 20 * time.Second
	maksCoba   = 3
)

type Config struct {
	// APIKey adalah kunci Resend (diawali "re_").
	APIKey string
	// From adalah pengirim, boleh "Nama <alamat@domain>".
	//
	// Domainnya harus sudah diverifikasi di Resend. Alamat di domain yang belum
	// terverifikasi ditolak dengan 403 — bukan diam-diam masuk spam, jadi
	// kegagalannya terlihat.
	From string
	// BaseURL menimpa alamat API. Diisi hanya dalam tes.
	BaseURL string
}

type Mailer struct {
	cfg Config
	hc  *http.Client
	log *slog.Logger
	// jeda adalah satuan jeda antar percobaan; dinolkan dalam tes.
	jeda time.Duration
}

func New(cfg Config) *Mailer {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" {
		cfg.BaseURL = baseBawaan
	}
	return &Mailer{
		cfg: cfg,
		hc:  &http.Client{Timeout: batasWaktu},
		// Logger bawaan proses, seperti httpx.Fail — main() sudah memanggil
		// slog.SetDefault sebelum apa pun disajikan.
		log:  slog.Default(),
		jeda: time.Second,
	}
}

// Siap melaporkan konfigurasinya lengkap.
//
// From ikut diwajibkan, tidak seperti pada versi SMTP yang bisa memakai
// username sebagai cadangan. Resend tidak punya cadangan: "from" wajib ada di
// badan permintaan, dan yang kosong ditolak saat kirim — jauh dari tempat
// orang bisa melihatnya.
func (m *Mailer) Siap() bool {
	return m != nil && m.cfg.APIKey != "" && alamatSaja(m.cfg.From) != ""
}

// Pesan adalah satu email teks-dan-HTML untuk satu penerima.
type Pesan struct {
	Ke     string
	Subjek string
	// Jenis adalah label pendek untuk log, mis. "otp" atau "reset-kata-sandi".
	//
	// ADA KARENA SUBJEK TIDAK BOLEH DICATAT. Subjek email OTP berbunyi
	// "Kode masuk 817810 — BNI Finance Hub": kodenya ada di dalamnya, supaya
	// terbaca dari pratinjau notifikasi tanpa membuka email. Mencatat subjek
	// berarti setiap kode masuk yang masih berlaku tersimpan apa adanya di log
	// aplikasi — yang dibaca lebih banyak orang, disimpan lebih lama, dan
	// sering dikirim ke pihak ketiga untuk dianalisis.
	//
	// Bahayanya baru muncul dari GABUNGAN dua perubahan yang masing-masing
	// masuk akal: satu menaruh kode di subjek, satu lagi mencatat subjek.
	// Keduanya digabung tanpa satu pun konflik teks.
	//
	// Label ini disediakan pemanggil, bukan diturunkan dari isi pesan. Yang
	// lupa mengisinya menghasilkan log tanpa jenis — kehilangan konteks, bukan
	// kebocoran.
	Jenis string
	// Teks selalu dikirim; HTML opsional.
	//
	// Keduanya, bukan salah satu: klien yang menolak HTML — dan penyaring spam
	// yang mencurigai email HTML tanpa alternatif teks — akan menampilkan
	// bagian teksnya. Email verifikasi yang berakhir di folder spam sama saja
	// dengan tidak terkirim.
	Teks string
	HTML string
}

type badanKirim struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text,omitempty"`
	HTML    string   `json:"html,omitempty"`
}

type jawaban struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Kirim mengirim satu pesan dan mencatat id-nya.
//
// ctx dipakai untuk batas waktu: panggilan jaringan yang menggantung akan
// menahan permintaan HTTP yang memanggilnya, dan pengguna melihat halaman yang
// membeku alih-alih pesan galat.
func (m *Mailer) Kirim(ctx context.Context, p Pesan) error {
	id, err := m.KirimDenganID(ctx, p)
	if err != nil {
		return err
	}
	// Yang dicatat hanya id dan jenis — bukan penerima, bukan subjek.
	//
	// id-nya cukup untuk menemukan pesan itu di dasbor Resend, lengkap dengan
	// alamat tujuan, subjek, dan status pengirimannya. Log aplikasi ini dibaca
	// lebih banyak orang daripada dasbor itu, jadi apa pun yang sudah ada di
	// sana tidak perlu diulang di sini.
	m.log.Info("email terkirim", "id", id, "jenis", bersih(p.Jenis))
	return nil
}

// KirimDenganID mengirim satu pesan dan mengembalikan id pesan dari Resend.
func (m *Mailer) KirimDenganID(ctx context.Context, p Pesan) (string, error) {
	if !m.Siap() {
		return "", ErrTidakDikonfigurasi
	}
	ke := bersih(p.Ke)
	if ke == "" {
		return "", errors.New("penerima kosong")
	}

	// Header dibersihkan dari CR/LF sebelum berangkat.
	//
	// Nilainya berasal dari data pengguna, dan Resend-lah yang merakit pesan
	// RFC 5322 di sisinya: baris baru yang lolos ke subjek atau alamat adalah
	// header tambahan yang kita titipkan — sebuah Bcc, misalnya, yang mengirim
	// salinan setiap tautan reset ke alamat lain. Membersihkannya di sini
	// berarti tidak bergantung pada perakit di seberang untuk menutupinya.
	badan, err := json.Marshal(badanKirim{
		From:    bersih(m.cfg.From),
		To:      []string{ke},
		Subject: bersih(p.Subjek),
		Text:    p.Teks,
		HTML:    p.HTML,
	})
	if err != nil {
		return "", fmt.Errorf("rakit permintaan: %w", err)
	}

	var terakhir error
	for coba := 1; coba <= maksCoba; coba++ {
		id, ulangi, err := m.sekaliKirim(ctx, badan)
		if err == nil {
			return id, nil
		}
		terakhir = err
		if !ulangi || coba == maksCoba {
			break
		}
		// Jeda naik: 1 detik lalu 2 detik. Batas Resend 2 permintaan/detik, dan
		// pengiriman massal (blast pengingat) menabraknya dengan mudah.
		select {
		case <-time.After(time.Duration(coba) * m.jeda):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return "", terakhir
}

// sekaliKirim melakukan satu percobaan. ulangi=true berarti permintaannya
// DIPASTIKAN tidak diproses, jadi mengulanginya tidak bisa menggandakan email.
func (m *Mailer) sekaliKirim(ctx context.Context, badan []byte) (id string, ulangi bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		m.cfg.BaseURL+"/emails", bytes.NewReader(badan))
	if err != nil {
		return "", false, fmt.Errorf("kirim email: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := m.hc.Do(req)
	if err != nil {
		// Galat transport TIDAK diulang.
		//
		// Ia ambigu: permintaannya mungkin sudah sampai dan jawabannya yang
		// hilang. Mengulangnya bisa mengirim email kedua ke orang yang sama —
		// dan pada email berisi tautan reset, dua tautan sah yang beredar lebih
		// buruk daripada satu email yang gagal dan bisa diminta ulang.
		return "", false, fmt.Errorf("kirim email: %w", err)
	}
	defer res.Body.Close()

	// Dibatasi: badan galat dari perantara (proxy, halaman status penyedia)
	// bisa berupa HTML sepanjang apa pun, dan seluruhnya berakhir di log.
	isi, _ := io.ReadAll(io.LimitReader(res.Body, 8<<10))

	var j jawaban
	_ = json.Unmarshal(isi, &j)

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		if j.ID == "" {
			// 2xx tanpa id bukan keberhasilan yang bisa ditelusuri. Melaporkannya
			// sebagai berhasil berarti mengarang bukti pengiriman.
			return "", false, fmt.Errorf("kirim email: jawaban %d tanpa id: %s", res.StatusCode, ringkas(isi))
		}
		return j.ID, false, nil
	}

	// 429 berarti Resend menolak SEBELUM memproses — mengulanginya aman.
	// 5xx tidak: pesannya mungkin sudah masuk antrean di sana.
	return "", res.StatusCode == http.StatusTooManyRequests,
		fmt.Errorf("kirim email: %s", pesanGalat(res.StatusCode, j, isi))
}

// pesanGalat mengutamakan penjelasan Resend sendiri.
//
// "The reddie.id domain is not verified" memberi tahu apa yang harus dilakukan;
// "403 Forbidden" tidak. Kode statusnya tetap ikut supaya bisa dicocokkan
// dengan dokumentasi.
func pesanGalat(status int, j jawaban, mentah []byte) string {
	if j.Message != "" {
		if j.Name != "" {
			return fmt.Sprintf("%d %s: %s", status, j.Name, j.Message)
		}
		return fmt.Sprintf("%d: %s", status, j.Message)
	}
	return fmt.Sprintf("%d: %s", status, ringkas(mentah))
}

func ringkas(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "(badan kosong)"
	}
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return bersih(s)
}

// bersih meratakan CR/LF jadi satu spasi.
//
// "\r\n" didaftarkan lebih dulu supaya pasangannya jadi SATU spasi; Replacer
// memilih pola yang terdaftar lebih awal pada posisi yang sama.
func bersih(v string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(strings.TrimSpace(v))
}

// alamatSaja mengupas "Nama <alamat@x>" menjadi "alamat@x", dan mengembalikan
// "" bila tidak ada alamat di dalamnya.
//
// Dipakai Siap() untuk menolak From yang tidak berisi alamat sama sekali —
// nilai seperti "BNI Finance Hub" lolos dari pemeriksaan "tidak kosong" tapi
// pasti ditolak Resend.
func alamatSaja(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.LastIndex(v, "<"); i >= 0 {
		if j := strings.Index(v[i:], ">"); j > 0 {
			v = strings.TrimSpace(v[i+1 : i+j])
		}
	}
	if !strings.Contains(v, "@") {
		return ""
	}
	return v
}
