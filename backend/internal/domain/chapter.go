package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Chapter mirrors the `chapters` table. It originates as a read-only mirror of
// BNI Visitor Management, but is writable here so a chapter can be corrected or
// added without waiting for a sync run.
type Chapter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	AreaName    *string   `json:"areaName,omitempty"`
	CityName    *string   `json:"cityName,omitempty"`
	SyncedAt    time.Time `json:"syncedAt"`
}

// CreateChapterInput is the POST body. ID may be omitted — the database then
// generates one, which is what you want for chapters created outside BNI VM.
type CreateChapterInput struct {
	ID          *string `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName"`
	AreaName    *string `json:"areaName"`
	CityName    *string `json:"cityName"`
}

func (in CreateChapterInput) Validate() error {
	if in.Name == "" {
		return fmt.Errorf("name wajib diisi")
	}
	return nil
}

type UpdateChapterInput struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"displayName"`
	AreaName    *string `json:"areaName"`
	CityName    *string `json:"cityName"`
}

func (in UpdateChapterInput) Validate() error {
	if in.Name != nil && *in.Name == "" {
		return fmt.Errorf("name tidak boleh kosong")
	}
	if in.DisplayName != nil && *in.DisplayName == "" {
		return fmt.Errorf("displayName tidak boleh kosong")
	}
	return nil
}

// ChapterFilter drives GET /chapters.
type ChapterFilter struct {
	Search   string
	CityName string
	AreaName string
	Limit    int
	Offset   int
}

// ChapterCounts adalah angka ringkas per chapter untuk halaman daftar chapter.
//
// Ada di server karena halaman itu dulu menghitungnya di klien dari seluruh
// member dan seluruh invoice — dua daftar yang sama-sama dipaku pada 200 baris,
// sehingga jumlah membernya berhenti bertambah di 200 dan tunggakannya hanya
// menjumlahkan 200 invoice pertama. Angka uang yang diam-diam terlalu kecil.
type ChapterCounts struct {
	ChapterID   string `json:"chapterId"`
	MemberCount int    `json:"memberCount"`
	// Outstanding adalah NOMINAL invoice sent + overdue, bukan cacahnya.
	Outstanding int64 `json:"outstanding"`
}

var bukanAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// AwalanChapter mendahului nama tampil setiap chapter.
//
// Nama organisasinya ditulis lengkap, bukan "BNI": singkatan itu juga dipakai
// Bank Negara Indonesia, dan situs yang menampilkannya di samping formulir
// kata sandi ditandai sebagai phishing.
const AwalanChapter = "Business Network International "

var awalanOrganisasi = regexp.MustCompile(`(?i)^(bni|business network international)\s+`)

// IntiChapter membuang nama organisasi dari depan nama chapter:
// "BNI Supreme" dan "Business Network International Supreme" menjadi "Supreme".
func IntiChapter(nama string) string {
	return strings.TrimSpace(awalanOrganisasi.ReplaceAllString(strings.TrimSpace(nama), ""))
}

// PanjangkanBNI menulis ulang nama tampil berawalan "BNI " dengan
// AwalanChapter. Nama tanpa awalan organisasi dikembalikan apa adanya: nama
// tampil bisa diubah admin, dan nama yang sengaja ditulis lain bukan untuk
// ditimpa.
func PanjangkanBNI(namaTampil string) string {
	inti := IntiChapter(namaTampil)
	if inti == "" || inti == strings.TrimSpace(namaTampil) {
		return namaTampil
	}
	return AwalanChapter + inti
}

// KunciChapter menyamakan penulisan nama chapter antarsumber: "Supreme",
// "BNI Supreme", "Business Network International Supreme", "BNI Supreme
// Chapter", dan "ch-supreme" berkunci sama.
func KunciChapter(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = IntiChapter(strings.TrimPrefix(s, "ch-"))
	s = bukanAlnum.ReplaceAllString(s, "")
	if inti := strings.TrimSuffix(s, "chapter"); inti != "" {
		return inti
	}
	return s
}
