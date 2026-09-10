// Package mailer mengirim email transaksional lewat SMTP.
//
// Hanya net/smtp dari pustaka standar — tidak ada dependensi baru. Yang
// dibutuhkan aplikasi ini sempit: beberapa pesan pendek, satu penerima, tanpa
// lampiran. Pustaka email penuh membawa parser MIME, antrean, dan template
// engine yang tidak satu pun dipakai di sini.
package mailer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// ErrTidakDikonfigurasi dikembalikan saat SMTP belum diisi.
//
// Dibedakan dari galat pengiriman supaya pemanggilnya bisa menjawab 503 dengan
// pesan yang jelas — "belum dikonfigurasi" adalah keadaan yang bisa diperbaiki
// orang, sedangkan "gagal kirim" belum tentu.
var ErrTidakDikonfigurasi = errors.New("SMTP belum dikonfigurasi")

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

type Mailer struct {
	cfg Config
	// kirim bisa diganti dalam tes; produksi memakai smtpKirim.
	kirim func(alamat string, auth smtp.Auth, dari string, ke []string, pesan []byte) error
}

func New(cfg Config) *Mailer {
	// From boleh kosong: alamat pengirim yang wajar adalah akun SMTP itu
	// sendiri, dan Gmail memang menolak From yang berbeda dari akunnya.
	if strings.TrimSpace(cfg.From) == "" {
		cfg.From = cfg.User
	}
	return &Mailer{cfg: cfg, kirim: smtp.SendMail}
}

// Siap melaporkan konfigurasinya lengkap.
func (m *Mailer) Siap() bool {
	return m != nil && m.cfg.Host != "" && m.cfg.Port != "" &&
		m.cfg.User != "" && m.cfg.Password != ""
}

// Pesan adalah satu email teks-dan-HTML untuk satu penerima.
type Pesan struct {
	Ke     string
	Subjek string
	// Teks selalu dikirim; HTML opsional.
	//
	// Keduanya, bukan salah satu: klien yang menolak HTML — dan penyaring spam
	// yang mencurigai email HTML tanpa alternatif teks — akan menampilkan
	// bagian teksnya. Email verifikasi yang berakhir di folder spam sama saja
	// dengan tidak terkirim.
	Teks string
	HTML string
}

// Kirim mengirim satu pesan.
//
// ctx dipakai untuk batas waktu: SMTP yang menggantung akan menahan permintaan
// HTTP yang memanggilnya, dan pengguna melihat halaman yang membeku alih-alih
// pesan galat.
func (m *Mailer) Kirim(ctx context.Context, p Pesan) error {
	if !m.Siap() {
		return ErrTidakDikonfigurasi
	}
	if strings.TrimSpace(p.Ke) == "" {
		return errors.New("penerima kosong")
	}

	badan := m.rakit(p)
	alamat := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)

	selesai := make(chan error, 1)
	go func() {
		selesai <- m.kirim(alamat, auth, alamatAmplop(m.cfg), []string{p.Ke}, badan)
	}()

	select {
	case err := <-selesai:
		if err != nil {
			return fmt.Errorf("kirim email: %w", err)
		}
		return nil
	case <-ctx.Done():
		// Goroutine-nya dibiarkan selesai sendiri; ia hanya memegang satu
		// koneksi dan akan berakhir pada timeout TCP. Membunuhnya di tengah
		// percakapan SMTP justru bisa meninggalkan pesan separuh terkirim.
		return ctx.Err()
	}
}

// alamatAmplop mengembalikan alamat pengirim untuk perintah MAIL FROM.
//
// DIAMBIL DARI From, BUKAN DARI USER — dan itu perbedaan yang baru terlihat
// setelah dicoba pada penyedia kedua.
//
// Versi pertama memakai cfg.User sebagai pengirim amplop. Pada Gmail itu
// kebetulan benar: username SMTP-nya memang alamat email. Pada Resend username
// SMTP-nya literal "resend", dan servernya menolak:
//
//	501 "Error: Bad sender address syntax"
//
// Diuji langsung terhadap Resend: amplop "onboarding@resend.dev" terkirim,
// amplop "resend" gagal. Satu penyedia menyembunyikan bug ini sepenuhnya karena
// dua nilai yang berbeda arti kebetulan sama isinya.
//
// From boleh berbentuk "Nama <alamat@x>" — yang masuk amplop hanya alamatnya;
// nama tampilannya milik header, bukan protokolnya.
func alamatAmplop(c Config) string {
	if a := alamatSaja(c.From); a != "" {
		return a
	}
	// Cadangan terakhir: sebagian penyedia memang memakai alamat sebagai
	// username. Kalau From kosong dan User pun bukan alamat, biarkan servernya
	// yang menolak dengan pesannya sendiri — menebak di sini hanya mengganti
	// satu galat jelas dengan galat lain yang membingungkan.
	return strings.TrimSpace(c.User)
}

// alamatSaja mengupas "Nama <alamat@x>" menjadi "alamat@x".
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

// rakit menyusun pesan RFC 5322. Header dibersihkan dari CR/LF.
//
// Tanpa itu, subjek atau alamat yang mengandung baris baru bisa MENYISIPKAN
// header tambahan — sebuah Bcc, misalnya, yang mengirim salinan setiap tautan
// reset ke alamat lain. Nilai-nilai ini berasal dari data pengguna, jadi
// kemungkinannya bukan teoretis.
func (m *Mailer) rakit(p Pesan) []byte {
	bersih := func(v string) string {
		// "\r\n" didaftarkan lebih dulu supaya pasangannya jadi SATU spasi;
		// Replacer memilih pola yang terdaftar lebih awal pada posisi yang sama.
		return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(strings.TrimSpace(v))
	}

	var b strings.Builder
	tulis := func(f string, v ...any) { fmt.Fprintf(&b, f+"\r\n", v...) }

	batas := fmt.Sprintf("batas-%d", time.Now().UnixNano())
	tulis("From: %s", bersih(m.cfg.From))
	tulis("To: %s", bersih(p.Ke))
	tulis("Subject: %s", bersih(p.Subjek))
	tulis("MIME-Version: 1.0")

	if p.HTML == "" {
		tulis("Content-Type: text/plain; charset=UTF-8")
		tulis("")
		b.WriteString(p.Teks)
		return []byte(b.String())
	}

	tulis("Content-Type: multipart/alternative; boundary=%q", batas)
	tulis("")
	tulis("--%s", batas)
	tulis("Content-Type: text/plain; charset=UTF-8")
	tulis("")
	b.WriteString(p.Teks + "\r\n")
	tulis("--%s", batas)
	tulis("Content-Type: text/html; charset=UTF-8")
	tulis("")
	b.WriteString(p.HTML + "\r\n")
	tulis("--%s--", batas)
	return []byte(b.String())
}
