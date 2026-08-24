package reminder

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

// stub meniru basis data di memori, termasuk sifat reminder_log yang paling
// penting: klaim kedua atas pasangan (invoice, offset) yang sama harus gagal.
type stub struct {
	set          Pengaturan
	kandidat     map[int][]Kandidat
	diklaim      map[string]bool
	dicatat      map[string]bool
	kunci        bool
	kunciDilepas int
}

func stubSiap() *stub {
	return &stub{
		set: Pengaturan{
			NotifikasiAktif: true, WorkerAktif: true,
			Offsets: []int{7, 3}, Interval: time.Hour,
		},
		kandidat: map[int][]Kandidat{
			7: {{InvoiceID: "inv-1", Number: "INV-1", Offset: 7}},
			3: {{InvoiceID: "inv-2", Number: "INV-2", Offset: 3}},
		},
		diklaim: map[string]bool{},
		dicatat: map[string]bool{},
		kunci:   true,
	}
}

func kunciDari(id string, offset int) string { return id + ":" + string(rune('0'+offset)) }

func (s *stub) Pengaturan(context.Context) (Pengaturan, error) { return s.set, nil }
func (s *stub) Kandidat(_ context.Context, offset, _ int) ([]Kandidat, error) {
	return s.kandidat[offset], nil
}
func (s *stub) Klaim(_ context.Context, id string, offset int) (bool, error) {
	k := kunciDari(id, offset)
	if s.diklaim[k] {
		return false, nil
	}
	s.diklaim[k] = true
	return true, nil
}
func (s *stub) Catat(_ context.Context, id string, offset int, _ bool, _ string) error {
	s.dicatat[kunciDari(id, offset)] = true
	return nil
}
func (s *stub) AmbilKunci(context.Context) (func(), bool, error) {
	if !s.kunci {
		return nil, false, nil
	}
	return func() { s.kunciDilepas++ }, true, nil
}

type pengirim struct {
	terkirim []string
	gagalkan bool
}

func (p *pengirim) Remind(_ context.Context, id string, _ RemindOptions) error {
	p.terkirim = append(p.terkirim, id)
	if p.gagalkan {
		return errors.New("Paper.id menolak")
	}
	return nil
}

func worker(s *stub, p *pengirim) *Worker {
	return NewWorker(s, p, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// SATU PENGINGAT TIDAK BOLEH TERKIRIM DUA KALI, bahkan bila workernya berputar
// berkali-kali.
//
// Ini sifat yang paling mahal bila salah: tiap pengiriman membakar nomor invoice
// Paper.id secara permanen, dan pesan yang telanjur sampai ke member tidak bisa
// ditarik kembali. Worker yang restart — deploy, crash, kontainer dijadwal ulang
// — melihat kembali seluruh invoice yang sama.
func TestPengingatTidakTerkirimDuaKali(t *testing.T) {
	s, p := stubSiap(), &pengirim{}
	w := worker(s, p)

	w.Putaran(context.Background())
	w.Putaran(context.Background())
	w.Putaran(context.Background())

	if len(p.terkirim) != 2 {
		t.Errorf("terkirim %d kali (%v), mau 2 — satu per invoice", len(p.terkirim), p.terkirim)
	}
}

// Diklaim SEBELUM dikirim: pengiriman yang GAGAL tetap meninggalkan klaim,
// sehingga putaran berikutnya tidak mencobanya lagi.
//
// Arahnya dipilih sadar. Kalau klaimnya dilepas saat gagal, kegagalan sementara
// dari Paper.id akan membuat worker mencoba lagi tiap putaran — dan setiap
// percobaan yang sebenarnya BERHASIL di sisi Paper.id tapi gagal terbaca di
// sisi kita menghasilkan pesan ganda ke member.
func TestKlaimBertahanMeskiPengirimanGagal(t *testing.T) {
	s, p := stubSiap(), &pengirim{gagalkan: true}
	w := worker(s, p)

	w.Putaran(context.Background())
	w.Putaran(context.Background())

	if len(p.terkirim) != 2 {
		t.Errorf("dicoba %d kali, mau 2 — kegagalan tidak boleh dicoba ulang otomatis", len(p.terkirim))
	}
	if len(s.dicatat) != 2 {
		t.Errorf("hasil tercatat %d, mau 2 — kegagalan harus tetap terekam", len(s.dicatat))
	}
}

// Kedua sakelar harus benar-benar menghentikan pengiriman.
func TestSakelarMatiMenghentikanPengiriman(t *testing.T) {
	kasus := map[string]func(*stub){
		"worker dimatikan":     func(s *stub) { s.set.WorkerAktif = false },
		"notifikasi dimatikan": func(s *stub) { s.set.NotifikasiAktif = false },
		"tanpa offset":         func(s *stub) { s.set.Offsets = nil },
	}
	for nama, matikan := range kasus {
		t.Run(nama, func(t *testing.T) {
			s, p := stubSiap(), &pengirim{}
			matikan(s)
			worker(s, p).Putaran(context.Background())
			if len(p.terkirim) != 0 {
				t.Errorf("%d pengingat terkirim padahal %s", len(p.terkirim), nama)
			}
		})
	}
}

// Tanpa kunci, worker tidak mengirim apa pun — replika lain sedang bekerja.
func TestTanpaKunciTidakMengirim(t *testing.T) {
	s, p := stubSiap(), &pengirim{}
	s.kunci = false
	worker(s, p).Putaran(context.Background())
	if len(p.terkirim) != 0 {
		t.Errorf("%d terkirim tanpa memegang kunci", len(p.terkirim))
	}
}

// Kunci WAJIB dilepas, termasuk saat putarannya berakhir normal.
//
// Session-scoped advisory lock yang tidak dilepas akan menahan SELURUH putaran
// berikutnya sampai prosesnya mati — worker berhenti bekerja tanpa satu pun
// pesan galat.
func TestKunciSelaluDilepas(t *testing.T) {
	s, p := stubSiap(), &pengirim{}
	w := worker(s, p)
	w.Putaran(context.Background())
	w.Putaran(context.Background())
	if s.kunciDilepas != 2 {
		t.Errorf("kunci dilepas %d kali dari 2 putaran", s.kunciDilepas)
	}
}

func TestBacaOffsets(t *testing.T) {
	kasus := map[string][]int{
		"7,3,1":       {7, 3, 1},
		" 7 , 3 ":     {7, 3},
		"7,7,3":       {7, 3}, // ganda dibuang
		"7,abc,3":     {7, 3}, // salah ketik tidak mematikan sisanya
		"0,-1,3":      {3},    // nol dan negatif bukan pengingat
		"400,3":       {3},    // di luar jangkauan wajar
		"":            nil,
		"tidak,valid": nil,
	}
	for masukan, mau := range kasus {
		got := bacaOffsets(masukan)
		if len(got) != len(mau) {
			t.Errorf("bacaOffsets(%q) = %v, mau %v", masukan, got, mau)
			continue
		}
		for i := range got {
			if got[i] != mau[i] {
				t.Errorf("bacaOffsets(%q) = %v, mau %v", masukan, got, mau)
				break
			}
		}
	}
}
