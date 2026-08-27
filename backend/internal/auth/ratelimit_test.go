package auth

import (
	"testing"
	"time"
)

func pembatasBeku(mulai time.Time) (*Pembatas, func(time.Duration)) {
	sekarang := mulai
	p := NewPembatas()
	p.now = func() time.Time { return sekarang }
	return p, func(d time.Duration) { sekarang = sekarang.Add(d) }
}

func TestMenguncisSesudahBatasKegagalan(t *testing.T) {
	p, _ := pembatasBeku(time.Now())

	for i := 0; i < MaksGagal-1; i++ {
		p.Gagal("orang@contoh.id")
		if _, terkunci := p.Terkunci("orang@contoh.id"); terkunci {
			t.Fatalf("terkunci terlalu cepat, setelah %d kegagalan", i+1)
		}
	}
	p.Gagal("orang@contoh.id")
	if _, terkunci := p.Terkunci("orang@contoh.id"); !terkunci {
		t.Errorf("tidak terkunci setelah %d kegagalan", MaksGagal)
	}
}

// Kunci harus MEMBUKA SENDIRI setelah jendelanya lewat.
//
// Kunci permanen yang butuh admin menjadikan gangguan sebagai senjata:
// siapa pun bisa mengunci akun orang lain dan memaksa admin turun tangan.
func TestKunciTerbukaSendiriSetelahJendela(t *testing.T) {
	p, maju := pembatasBeku(time.Now())
	for i := 0; i < MaksGagal; i++ {
		p.Gagal("orang@contoh.id")
	}
	if _, terkunci := p.Terkunci("orang@contoh.id"); !terkunci {
		t.Fatal("seharusnya terkunci")
	}

	maju(JendelaGagal + time.Second)
	if _, terkunci := p.Terkunci("orang@contoh.id"); terkunci {
		t.Error("masih terkunci setelah jendelanya lewat")
	}
}

// Sisa waktu yang dilaporkan harus JUJUR — dihitung dari kegagalan tertua yang
// masih di dalam jendela, bukan yang terbaru.
//
// Versi pertama komentar saya mengklaim memakai yang terbaru akan menahan akun
// terkunci selamanya. Itu keliru: bersihkan() sudah memangkas ke jendela, jadi
// kuncinya tetap terbuka sendiri. Yang benar-benar berbeda adalah angka
// Retry-After — dan petunjuk waktu yang meleset membuat orang menunggu lebih
// lama daripada perlu.
func TestSisaWaktuDihitungDariKegagalanTertua(t *testing.T) {
	p, maju := pembatasBeku(time.Now())

	// Sepuluh kegagalan yang tersebar: yang pertama jauh lebih tua.
	p.Gagal("orang@contoh.id")
	maju(JendelaGagal / 2)
	for i := 0; i < MaksGagal-1; i++ {
		p.Gagal("orang@contoh.id")
	}

	sisa, terkunci := p.Terkunci("orang@contoh.id")
	if !terkunci {
		t.Fatal("seharusnya terkunci")
	}
	// Kegagalan tertua sudah berumur setengah jendela, jadi sisanya ~setengah.
	// Kalau dihitung dari yang terbaru, sisanya akan mendekati satu jendela
	// penuh — dua kali lipat dari yang sebenarnya.
	if sisa > JendelaGagal*3/4 {
		t.Errorf("sisa = %v, terlalu lama — dihitung dari kegagalan terbaru?", sisa)
	}
}

// Kunci tetap terbuka sendiri meski percobaan terus berdatangan.
func TestKunciTetapTerbukaMeskiTerusDicoba(t *testing.T) {
	p, maju := pembatasBeku(time.Now())
	for i := 0; i < MaksGagal; i++ {
		p.Gagal("orang@contoh.id")
	}
	for i := 0; i < 20; i++ {
		maju(JendelaGagal / 25)
		p.Gagal("orang@contoh.id")
	}
	maju(JendelaGagal + time.Second)

	if _, terkunci := p.Terkunci("orang@contoh.id"); terkunci {
		t.Error("masih terkunci setelah seluruh kegagalan keluar jendela")
	}
}

// Login yang berhasil membuktikan pemiliknya orang yang benar.
func TestBerhasilMenghapusHitungan(t *testing.T) {
	p, _ := pembatasBeku(time.Now())
	for i := 0; i < MaksGagal-1; i++ {
		p.Gagal("orang@contoh.id")
	}
	p.Berhasil("orang@contoh.id")

	for i := 0; i < MaksGagal-1; i++ {
		p.Gagal("orang@contoh.id")
		if _, terkunci := p.Terkunci("orang@contoh.id"); terkunci {
			t.Fatalf("terkunci setelah %d kegagalan — hitungan lama tidak terhapus", i+1)
		}
	}
}

// Mengunci satu akun tidak boleh menyentuh akun lain.
func TestKunciTidakMenularKeAkunLain(t *testing.T) {
	p, _ := pembatasBeku(time.Now())
	for i := 0; i < MaksGagal*2; i++ {
		p.Gagal("korban@contoh.id")
	}
	if _, terkunci := p.Terkunci("orang-lain@contoh.id"); terkunci {
		t.Error("akun lain ikut terkunci")
	}
}

func TestEmailTidakPekaHurufBesarDanSpasi(t *testing.T) {
	p, _ := pembatasBeku(time.Now())
	for i := 0; i < MaksGagal; i++ {
		p.Gagal("  Orang@Contoh.ID  ")
	}
	if _, terkunci := p.Terkunci("orang@contoh.id"); !terkunci {
		t.Error("huruf besar atau spasi bisa dipakai melewati batas")
	}
}

// PETA TIDAK BOLEH TUMBUH TANPA BATAS.
//
// Kuncinya email, jadi penyerang yang mengirim sejuta email berbeda menumbuhkan
// peta ini sampai prosesnya kehabisan memori — pembatas yang justru menjadi cara
// mematikan layanan.
func TestPetaBerbatasMeskiEmailSelaluBaru(t *testing.T) {
	p, _ := pembatasBeku(time.Now())
	for i := 0; i < maksEntri*3; i++ {
		p.Gagal(string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)) + "-" +
			time.Duration(i).String() + "@contoh.id")
	}
	p.mu.Lock()
	n := len(p.gagal)
	p.mu.Unlock()

	if n > maksEntri {
		t.Errorf("peta tumbuh sampai %d entri, batasnya %d", n, maksEntri)
	}
}
