package auth

import (
	"strings"
	"sync"
	"time"
)

// Pembatas percobaan masuk.
//
// Sebelum ini login tidak dibatasi sama sekali: menebak kata sandi bisa
// dilakukan secepat jaringan mengizinkan, dan MinPasswordLength hanya enam
// karakter. Enumerasi akun memang sudah dijaga — pesan galat untuk email tak
// dikenal dan sandi salah sengaja sama — tapi itu tidak memperlambat penebakan
// sama sekali.
//
// DIKUNCI PER EMAIL, BUKAN PER IP, dan itu keputusan yang menentukan.
//
// Backend ini berjalan di belakang Cloudflare. RemoteAddr karena itu berisi
// alamat Cloudflare, sama untuk SETIAP pengguna — membatasi per RemoteAddr
// berarti sepuluh kegagalan siapa pun mengunci seluruh orang di organisasi.
// Header CF-Connecting-IP bisa dibaca, tapi header bisa dipalsukan oleh siapa
// pun yang menembak origin langsung, jadi mempercayainya tanpa syarat justru
// membuat pembatasnya bisa dilewati.
//
// Email selalu ada di permintaan login dan tidak bisa dipalsukan untuk
// menghindari batas: menyerang satu akun berarti memakai email akun itu.
//
// Konsekuensi yang diterima sadar: seseorang bisa sengaja mengunci akun orang
// lain dengan sepuluh percobaan salah. Karena itu kuncinya BERBATAS WAKTU dan
// pendek — mengganggu, tapi jauh lebih ringan daripada membiarkan penebakan
// berjalan tanpa hambatan. Kunci permanen yang butuh admin justru menjadikan
// gangguan itu senjata.

const (
	// MaksGagal adalah jumlah kegagalan sebelum akun dikunci sementara.
	MaksGagal = 10
	// JendelaGagal adalah rentang waktu penghitungannya.
	JendelaGagal = 15 * time.Minute
	// maksEntri membatasi jumlah email yang dilacak sekaligus.
	//
	// Tanpa ini, penyerang yang mengirim sejuta email berbeda menumbuhkan peta
	// ini tanpa batas sampai prosesnya kehabisan memori — pembatas yang justru
	// menjadi cara mematikan layanan.
	maksEntri = 10_000
)

// Pembatas melacak kegagalan login per email.
type Pembatas struct {
	mu    sync.Mutex
	gagal map[string][]time.Time
	now   func() time.Time
}

func NewPembatas() *Pembatas {
	return &Pembatas{gagal: make(map[string][]time.Time), now: time.Now}
}

// Terkunci melaporkan email ini sedang dikunci, beserta sisa waktunya.
func (p *Pembatas) Terkunci(email string) (time.Duration, bool) {
	k := kunciEmail(email)
	if k == "" {
		return 0, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	sisa := p.bersihkan(k)
	if len(sisa) < MaksGagal {
		return 0, false
	}
	// Sisa waktu dihitung dari kegagalan TERTUA yang masih di dalam jendela,
	// karena itulah yang pertama akan gugur dan membuat hitungannya turun di
	// bawah batas.
	//
	// Memakai yang terbaru TIDAK menahan akun selamanya — bersihkan() sudah
	// memangkas ke jendela, jadi kuncinya tetap terbuka sendiri. Yang berbeda
	// hanya angka yang dilaporkan lewat Retry-After: memakai yang terbaru
	// menghasilkan tunggu yang lebih lama daripada kenyataannya, dan petunjuk
	// waktu yang meleset membuat orang menunggu lebih lama tanpa alasan.
	return JendelaGagal - p.now().Sub(sisa[0]), true
}

// Gagal mencatat satu percobaan yang salah.
func (p *Pembatas) Gagal(email string) {
	k := kunciEmail(email)
	if k == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.bersihkan(k)
	if len(p.gagal) >= maksEntri {
		if _, sudahAda := p.gagal[k]; !sudahAda {
			p.pangkas()
		}
	}
	p.gagal[k] = append(p.gagal[k], p.now())
}

// Berhasil menghapus catatan kegagalan.
//
// Login yang berhasil membuktikan pemiliknya memang orang yang benar, jadi
// menahannya karena percobaan sebelumnya tidak masuk akal.
func (p *Pembatas) Berhasil(email string) {
	k := kunciEmail(email)
	if k == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.gagal, k)
}

// bersihkan membuang kegagalan yang sudah keluar jendela. Pemanggil memegang mu.
func (p *Pembatas) bersihkan(k string) []time.Time {
	batas := p.now().Add(-JendelaGagal)
	sisa := p.gagal[k][:0]
	for _, t := range p.gagal[k] {
		if t.After(batas) {
			sisa = append(sisa, t)
		}
	}
	if len(sisa) == 0 {
		delete(p.gagal, k)
		return nil
	}
	p.gagal[k] = sisa
	return sisa
}

// pangkas membuang entri yang seluruh kegagalannya sudah kedaluwarsa.
//
// Dipanggil hanya saat peta menyentuh batas, bukan berkala: tanpa penyerang,
// peta ini berisi segelintir entri dan pemangkasan berkala hanya membakar CPU
// untuk pekerjaan yang tidak ada. Pemanggil memegang mu.
func (p *Pembatas) pangkas() {
	batas := p.now().Add(-JendelaGagal)
	for k, ts := range p.gagal {
		if len(ts) == 0 || ts[len(ts)-1].Before(batas) {
			delete(p.gagal, k)
		}
	}
	// Masih penuh berarti seluruh entrinya masih aktif — serangan sedang
	// berlangsung. Peta dikosongkan sepenuhnya alih-alih dibiarkan tumbuh:
	// kehilangan hitungan sesaat jauh lebih ringan daripada proses yang mati,
	// dan penyerangnya harus mengulang dari nol juga.
	if len(p.gagal) >= maksEntri {
		p.gagal = make(map[string][]time.Time)
	}
}

func kunciEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
