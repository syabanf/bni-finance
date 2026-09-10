package domain

import (
	"fmt"
	"time"
)

type MemberStatus string

const (
	MemberActive   MemberStatus = "active"
	MemberInactive MemberStatus = "inactive"
	MemberPending  MemberStatus = "pending"
	// MemberVisitor adalah tamu yang datang ke pertemuan tapi BELUM mendaftar.
	//
	// Ia dicatat supaya bisa diundang lagi dan diikuti sampai menjadi anggota.
	// Yang tidak boleh terjadi: ia ikut tertagih. Lihat BolehDitagih.
	MemberVisitor MemberStatus = "visitor"
)

// SemuaStatusMember adalah daftar lengkapnya, dan satu-satunya sumbernya.
//
// Valid() dan tes klasifikasi membacanya dari sini, jadi status yang
// ditambahkan tanpa memutuskan perlakuannya akan memerahkan tes — bukan
// diam-diam ikut aturan yang kebetulan berlaku.
var SemuaStatusMember = []MemberStatus{
	MemberActive, MemberInactive, MemberPending, MemberVisitor,
}

func (s MemberStatus) Valid() bool {
	for _, v := range SemuaStatusMember {
		if s == v {
			return true
		}
	}
	return false
}

// BolehDitagih melaporkan orang berstatus ini boleh menerima tagihan
// keanggotaan.
//
// DITULIS SEBAGAI SWITCH YANG MENYEBUT SETIAP STATUS, bukan sebagai
// `!= MemberVisitor`. Bentuk negatif membuat status yang ditambahkan nanti
// otomatis ikut tertagih — diam-diam, tanpa seorang pun memutuskannya. Di sini
// yang tidak disebut jatuh ke `false`, dan tes memaksa setiap status punya
// keputusan tertulis.
//
// Arah gagalnya dipilih sengaja: tagihan yang tidak terbit hanya perlu
// diperbaiki; tagihan yang terlanjur terkirim ke orang yang tidak pernah
// menjadi anggota tidak bisa ditarik kembali dari kotak masuknya.
//
// `inactive` dan `pending` tetap boleh, seperti sebelumnya — member yang
// keanggotaannya lewat masih ditagih perpanjangan, dan yang pending memang
// sedang menunggu tagihan pendaftarannya.
func (s MemberStatus) BolehDitagih() bool {
	switch s {
	case MemberActive, MemberInactive, MemberPending:
		return true
	case MemberVisitor:
		return false
	}
	return false
}

// Member mirrors the `members` table. Chapter is populated from a LEFT JOIN so
// list views don't need a second round-trip — same shape as the frontend's
// MemberWithChapter.
type Member struct {
	ID            string       `json:"id"`
	ChapterID     string       `json:"chapterId"`
	Name          string       `json:"name"`
	Email         *string      `json:"email,omitempty"`
	Phone         *string      `json:"phone,omitempty"`
	Company       *string      `json:"company,omitempty"`
	BusinessField *string      `json:"businessField,omitempty"`
	Status        MemberStatus `json:"status"`

	// Nullable in the schema, so these are `null` rather than absent — the
	// TypeScript type is `string | null`.
	JoinedDate  *Date `json:"joinedDate"`
	RenewalDate *Date `json:"renewalDate"`

	SyncedAt time.Time `json:"syncedAt"`

	Chapter *Chapter `json:"chapter,omitempty"`
}

// RenewalDueMember is a member whose membership is about to lapse.
type RenewalDueMember struct {
	Member
	DaysUntilDue int `json:"daysUntilDue"`
}

type CreateMemberInput struct {
	ID            *string       `json:"id"`
	ChapterID     string        `json:"chapterId"`
	Name          string        `json:"name"`
	Email         *string       `json:"email"`
	Phone         *string       `json:"phone"`
	Company       *string       `json:"company"`
	BusinessField *string       `json:"businessField"`
	Status        *MemberStatus `json:"status"`
	JoinedDate    *Date         `json:"joinedDate"`
	RenewalDate   *Date         `json:"renewalDate"`
}

func (in CreateMemberInput) Validate() error {
	switch {
	case in.ChapterID == "":
		return fmt.Errorf("chapterId wajib diisi")
	case in.Name == "":
		return fmt.Errorf("name wajib diisi")
	case in.Status != nil && !in.Status.Valid():
		return fmt.Errorf("status harus 'active', 'inactive', 'pending', atau 'visitor'")
	}
	return nil
}

type UpdateMemberInput struct {
	ChapterID     *string       `json:"chapterId"`
	Name          *string       `json:"name"`
	Email         *string       `json:"email"`
	Phone         *string       `json:"phone"`
	Company       *string       `json:"company"`
	BusinessField *string       `json:"businessField"`
	Status        *MemberStatus `json:"status"`
	JoinedDate    *Date         `json:"joinedDate"`
	RenewalDate   *Date         `json:"renewalDate"`
}

func (in UpdateMemberInput) Validate() error {
	switch {
	case in.Name != nil && *in.Name == "":
		return fmt.Errorf("name tidak boleh kosong")
	case in.ChapterID != nil && *in.ChapterID == "":
		return fmt.Errorf("chapterId tidak boleh kosong")
	case in.Status != nil && !in.Status.Valid():
		return fmt.Errorf("status harus 'active', 'inactive', 'pending', atau 'visitor'")
	}
	return nil
}

// MemberFilter drives GET /members.
type MemberFilter struct {
	ChapterID   string
	Status      string
	Search      string
	RenewalFrom string
	RenewalTo   string
	Limit       int
	Offset      int
}
