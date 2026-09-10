package mailer

import (
	"context"
	"net/smtp"
	"strings"
	"testing"
)

func mailerPalsu() (*Mailer, *[]byte) {
	var terkirim []byte
	m := New(Config{Host: "h", Port: "587", User: "u@x.test", Password: "p", From: "Pengirim <u@x.test>"})
	m.kirim = func(_ string, _ smtp.Auth, _ string, _ []string, pesan []byte) error {
		terkirim = pesan
		return nil
	}
	return m, &terkirim
}

// HEADER TIDAK BOLEH BISA DISUNTIK.
//
// Subjek dan alamat penerima berasal dari data pengguna. Baris baru di
// dalamnya bisa menyisipkan header tambahan — sebuah Bcc, misalnya, yang
// mengirim salinan SETIAP tautan reset kata sandi ke alamat penyerang.
// Kemungkinannya bukan teoretis: nama dan email diisi lewat formulir.
func TestHeaderTidakBisaDisuntik(t *testing.T) {
	m, terkirim := mailerPalsu()

	err := m.Kirim(context.Background(), Pesan{
		Ke:     "korban@contoh.test\r\nBcc: penyerang@jahat.test",
		Subjek: "Reset\r\nBcc: penyerang@jahat.test",
		Teks:   "isi",
	})
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}

	pesan := string(*terkirim)

	// Yang berbahaya adalah baris yang DIMULAI dengan nama header — itulah yang
	// dibaca server SMTP sebagai header. "Bcc:" yang muncul di tengah nilai
	// header lain hanyalah teks, dan melarangnya akan membuat tes ini menuduh
	// kode yang sebenarnya sudah benar. (Asersi pertama saya melakukan persis
	// itu, dan memerah pada keluaran yang aman.)
	for _, baris := range strings.Split(pesan, "\r\n") {
		if strings.HasPrefix(strings.ToLower(baris), "bcc:") {
			t.Errorf("header Bcc tersuntik sebagai baris tersendiri:\n%s", pesan)
		}
	}
	// Isinya tidak boleh hilang — hanya baris barunya yang diratakan jadi spasi,
	// supaya jejak percobaannya masih terbaca di header yang terkirim.
	if !strings.Contains(pesan, "Subject: Reset Bcc: penyerang@jahat.test") {
		t.Errorf("baris barunya seharusnya jadi satu spasi, bukan dibuang:\n%s", pesan)
	}
}

// Email teks-saja tidak boleh dibungkus multipart — penyaring spam
// mencurigainya, dan sebagian klien menampilkan penanda batasnya sebagai teks.
func TestTeksSajaTidakMultipart(t *testing.T) {
	m, terkirim := mailerPalsu()
	if err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "halo"}); err != nil {
		t.Fatalf("kirim: %v", err)
	}
	pesan := string(*terkirim)
	if strings.Contains(pesan, "multipart/alternative") {
		t.Error("pesan teks-saja dibungkus multipart")
	}
	if !strings.Contains(pesan, "Content-Type: text/plain; charset=UTF-8") {
		t.Error("Content-Type teks hilang")
	}
}

// HTML selalu disertai versi teks.
//
// Klien yang menolak HTML — dan penyaring spam yang mencurigai HTML tanpa
// alternatif — akan menampilkan bagian teksnya. Email verifikasi yang berakhir
// di folder spam sama saja dengan tidak terkirim.
func TestHTMLSelaluDisertaiTeks(t *testing.T) {
	m, terkirim := mailerPalsu()
	err := m.Kirim(context.Background(), Pesan{
		Ke: "a@b.test", Subjek: "s", Teks: "versi teks", HTML: "<p>versi html</p>",
	})
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}
	pesan := string(*terkirim)
	for _, harus := range []string{"multipart/alternative", "text/plain", "text/html", "versi teks", "versi html"} {
		if !strings.Contains(pesan, harus) {
			t.Errorf("bagian %q tidak ada di pesan", harus)
		}
	}
}

// Konfigurasi setengah jadi harus ditolak SEBELUM menyentuh jaringan, dengan
// galat yang bisa dibedakan dari kegagalan kirim.
func TestBelumDikonfigurasiDitolakJelas(t *testing.T) {
	m := New(Config{Host: "h"}) // user & password kosong
	if m.Siap() {
		t.Fatal("Siap() true padahal kredensialnya kosong")
	}
	err := m.Kirim(context.Background(), Pesan{Ke: "a@b.test", Subjek: "s", Teks: "t"})
	if err == nil {
		t.Fatal("kirim berhasil padahal SMTP belum dikonfigurasi")
	}
	if !strings.Contains(err.Error(), "belum dikonfigurasi") {
		t.Errorf("galat = %v, seharusnya menyebut belum dikonfigurasi", err)
	}
}
