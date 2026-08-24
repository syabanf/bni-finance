package importer

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// MaksUnggah membatasi ukuran berkas yang diterima.
//
// 10 MB sudah memuat puluhan ribu baris member. Batasnya ada bukan karena
// berkas besar itu salah, melainkan karena tanpa batas, satu unggahan bisa
// menghabiskan memori seluruh proses — mematikan layanan bagi semua orang,
// bukan hanya bagi yang mengunggah.
const MaksUnggah = 10 << 20

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register memasang rute impor.
//
// ADMIN SAJA. Impor menulis lintas chapter sekaligus — mengizinkan ST berarti
// memberi jalan mengubah data chapter lain lewat satu berkas, melewati seluruh
// batas yang dijaga internal/scope.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/import/{jenis}", auth.RequireAdmin(h.jalankan))
}

func (h *Handler) jalankan(w http.ResponseWriter, r *http.Request) {
	jenis := Jenis(strings.ToLower(r.PathValue("jenis")))
	if jenis != JenisChapter && jenis != JenisMember {
		httpx.Fail(w, httpx.BadRequest(
			fmt.Sprintf("jenis %q tidak dikenal — pakai 'chapters' atau 'members'", jenis)))
		return
	}

	// terapkan=false adalah BAWAAN, dan itu keputusan keamanan.
	//
	// Klien yang lupa menyertakan parameternya mendapat pratinjau, bukan
	// penulisan. Arah kebalikannya berarti sebuah panggilan yang salah ketik
	// menimpa data keanggotaan tanpa ada yang sempat melihatnya lebih dulu.
	terapkan := r.URL.Query().Get("terapkan") == "true"

	data, err := bacaUnggahan(r)
	if err != nil {
		httpx.Fail(w, err)
		return
	}

	hasil, err := h.svc.Jalankan(r.Context(), jenis, data, terapkan)
	if err != nil {
		// Galat dari importer hampir selalu kesalahan BERKASNYA — kolom wajib
		// hilang, format tidak terbaca — jadi 400, bukan 500. Menjawab 500
		// menyuruh orang mencari kerusakan server padahal yang perlu diperbaiki
		// adalah berkas di tangannya.
		if httpx.StatusOf(err) == http.StatusInternalServerError {
			httpx.Fail(w, httpx.BadRequest(err.Error()))
			return
		}
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, hasil)
}

// bacaUnggahan menerima berkas lewat multipart maupun body mentah.
//
// Keduanya diterima karena keduanya benar-benar dipakai: peramban mengirim
// multipart dari sebuah form, sedangkan curl dan skrip lebih mudah mengirim
// body mentah. Menuntut salah satunya saja membuat separuh pemakai gagal dengan
// pesan yang tidak menjelaskan apa pun.
func bacaUnggahan(r *http.Request) ([]byte, error) {
	// MaxBytesReader membatasi di lapisan yang benar: pembacaannya berhenti
	// pada batas, alih-alih membaca seluruh berkas ke memori dulu baru menolak.
	r.Body = http.MaxBytesReader(nil, r.Body, MaksUnggah+1024)

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(MaksUnggah); err != nil {
			return nil, httpx.BadRequest("berkas terlalu besar atau form tidak terbaca")
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			return nil, httpx.BadRequest("tidak ada berkas pada field 'file'")
		}
		defer f.Close()
		return bacaBatas(f)
	}
	return bacaBatas(r.Body)
}

func bacaBatas(rd io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(rd, MaksUnggah+1))
	if err != nil {
		return nil, httpx.BadRequest("berkas tidak bisa dibaca: " + err.Error())
	}
	if len(data) > MaksUnggah {
		return nil, httpx.BadRequest(fmt.Sprintf("berkas melebihi %d MB", MaksUnggah>>20))
	}
	if len(data) == 0 {
		return nil, httpx.BadRequest("berkas kosong")
	}
	return data, nil
}
