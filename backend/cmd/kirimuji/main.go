// Command kirimuji mengirim satu email percobaan memakai konfigurasi yang
// sedang berlaku, lalu mencetak id pesannya — atau alasan penolakannya.
//
// Ada karena pertanyaan "kenapa emailnya tidak sampai" punya terlalu banyak
// jawaban yang tidak bisa dibedakan dari luar: kunci salah, domain belum
// diverifikasi, MAIL_FROM tidak berisi alamat, atau emailnya memang terkirim
// dan berakhir di folder spam. Menebaknya lewat halaman "lupa kata sandi"
// mustahil — halaman itu SENGAJA menjawab hal yang sama untuk setiap keadaan,
// supaya tidak bisa dipakai memeriksa keanggotaan.
//
// Perkakas ini menjawab semuanya dalam satu baris:
//
//	go run ./cmd/kirimuji orang@contoh.com
//
// Berhasil  → id pesan, yang bisa dicari di https://resend.com/emails
// Gagal     → pesan Resend apa adanya, mis. "The X domain is not verified"
//
// Dijalankan dari direktori yang memuat .env, sama seperti server.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/mailer"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "pemakaian: go run ./cmd/kirimuji <alamat-tujuan>")
		os.Exit(2)
	}
	tujuan := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		// Load() mewajibkan DATABASE_URL, yang tidak dipakai di sini. Disebutkan
		// supaya orang tahu ini bukan soal emailnya.
		fmt.Fprintf(os.Stderr, "konfigurasi: %v\n", err)
		os.Exit(1)
	}

	surat := mailer.New(mailer.Config{APIKey: cfg.ResendAPIKey, From: cfg.MailFrom})
	fmt.Printf("MAIL_FROM      : %s\n", tampil(cfg.MailFrom))
	fmt.Printf("RESEND_API_KEY : %s\n", samar(cfg.ResendAPIKey))
	fmt.Printf("APP_BASE_URL   : %s\n", tampil(cfg.AppBaseURL))

	if !surat.Siap() {
		fmt.Fprintln(os.Stderr, "\nBELUM SIAP — RESEND_API_KEY dan MAIL_FROM harus terisi,")
		fmt.Fprintln(os.Stderr, "dan MAIL_FROM harus memuat alamat email (boleh \"Nama <a@b>\").")
		if cfg.SMTPUsang {
			fmt.Fprintln(os.Stderr, "\nSMTP_* masih terpasang tapi tidak dipakai lagi:")
			fmt.Fprintln(os.Stderr, "  SMTP_PASSWORD → RESEND_API_KEY")
			fmt.Fprintln(os.Stderr, "  SMTP_FROM     → MAIL_FROM")
		}
		os.Exit(1)
	}

	ctx, batal := context.WithTimeout(context.Background(), 30*time.Second)
	defer batal()

	id, err := surat.KirimDenganID(ctx, mailer.Pesan{
		Ke:     tujuan,
		Subjek: "Uji kirim — BNI Finance Hub",
		Teks: "Email ini dikirim oleh perkakas uji BNI Finance Hub.\n\n" +
			"Kalau ia sampai, tautan reset kata sandi dan kode OTP juga akan sampai\n" +
			"ke alamat ini.",
		HTML: `<p>Email ini dikirim oleh perkakas uji <b>BNI Finance Hub</b>.</p>` +
			`<p>Kalau ia sampai, tautan reset kata sandi dan kode OTP juga akan sampai ke alamat ini.</p>`,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGAGAL: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nTERKIRIM ke %s\n", tujuan)
	fmt.Printf("id pesan: %s\n", id)
	fmt.Println("Telusuri statusnya di https://resend.com/emails")
	fmt.Println("Belum masuk juga setelah beberapa menit? Periksa folder spam —")
	fmt.Println("Resend sudah menerimanya, jadi sisanya ada di sisi penerima.")
}

func tampil(v string) string {
	if v == "" {
		return "(kosong)"
	}
	return v
}

// samar menunjukkan kuncinya TERISI tanpa mencetak isinya: keluaran perkakas
// ini sering ditempel ke chat saat orang minta bantuan.
func samar(v string) string {
	if v == "" {
		return "(kosong)"
	}
	if len(v) <= 8 {
		return "(terisi)"
	}
	return v[:5] + "…" + v[len(v)-2:] + fmt.Sprintf(" (%d karakter)", len(v))
}
