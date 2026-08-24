package domain

import (
	"fmt"
	"strings"
	"time"
)

// Konfirmasi renewal — ST menanyakan, MC menjawab.
//
// Alurnya: ST membuka daftar member yang keanggotaannya segera jatuh tempo dan
// meminta konfirmasi. MC chapter itu menjawab per member. ST menerbitkan invoice
// hanya untuk yang menjawab akan memperpanjang.
//
// Yang dijaga di sini adalah hal yang tidak bisa dijaga basis data: siapa boleh
// menjawab, jawaban apa yang sah, dan kapan sebuah permintaan boleh berubah.

// RenewalAnswer adalah jawaban MC atas satu permintaan.
type RenewalAnswer string

const (
	RenewalPending   RenewalAnswer = "pending"
	RenewalWillRenew RenewalAnswer = "will_renew"
	RenewalWillNot   RenewalAnswer = "will_not"
	RenewalUnsure    RenewalAnswer = "unsure"
)

func (a RenewalAnswer) Valid() bool {
	switch a {
	case RenewalPending, RenewalWillRenew, RenewalWillNot, RenewalUnsure:
		return true
	}
	return false
}

// Terjawab melaporkan MC sudah memberi jawaban.
func (a RenewalAnswer) Terjawab() bool { return a != "" && a != RenewalPending }

// RenewalRequest mencerminkan satu baris renewal_requests.
type RenewalRequest struct {
	ID        string `json:"id"`
	MemberID  string `json:"memberId"`
	ChapterID string `json:"chapterId"`
	// Period adalah TAHUN keanggotaan yang ditanyakan, bukan tanggal ST
	// menanyakannya. Itu yang membedakan satu permintaan dari permintaan tahun
	// berikutnya.
	Period string `json:"period"`

	RequestedBy *string   `json:"requestedBy,omitempty"`
	RequestedAt time.Time `json:"requestedAt"`
	AssignedMC  *string   `json:"assignedMc,omitempty"`

	Answer     RenewalAnswer `json:"answer"`
	AnsweredBy *string       `json:"answeredBy,omitempty"`
	AnsweredAt *time.Time    `json:"answeredAt,omitempty"`
	Note       *string       `json:"note,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Diisi lewat join, supaya daftar tugas MC bisa ditampilkan tanpa
	// permintaan tambahan per baris.
	MemberName  *string `json:"memberName,omitempty"`
	ChapterName *string `json:"chapterName,omitempty"`
	RenewalDate *Date   `json:"renewalDate,omitempty"`
}

// CreateRenewalRequestInput adalah permintaan ST untuk sekumpulan member.
type CreateRenewalRequestInput struct {
	MemberIDs []string `json:"memberIds"`
	Period    string   `json:"period"`
	// AssignedMC boleh kosong; permintaan tetap terlihat oleh seluruh MC di
	// chapter itu. Menuntutnya terisi akan menghentikan ST yang chapternya
	// belum punya akun MC sama sekali.
	AssignedMC *string `json:"assignedMc"`
}

func (in CreateRenewalRequestInput) Validate() error {
	switch {
	case len(in.MemberIDs) == 0:
		return fmt.Errorf("memberIds wajib diisi")
	case len(in.MemberIDs) > MaxRenewalRequestBatch:
		return fmt.Errorf("terlalu banyak member dalam satu permintaan (maksimal %d)",
			MaxRenewalRequestBatch)
	case strings.TrimSpace(in.Period) == "":
		return fmt.Errorf("period wajib diisi")
	}
	return nil
}

// MaxRenewalRequestBatch membatasi satu permintaan.
//
// Bukan aturan bisnis, melainkan pagar: permintaan yang memuat seluruh basis
// data akan menghasilkan satu transaksi raksasa dan daftar tugas MC yang tidak
// mungkin dikerjakan. Chapter terbesar pun jauh di bawah angka ini.
const MaxRenewalRequestBatch = 500

// AnswerRenewalInput adalah jawaban MC.
type AnswerRenewalInput struct {
	Answer RenewalAnswer `json:"answer"`
	Note   *string       `json:"note"`
}

func (in AnswerRenewalInput) Validate() error {
	if !in.Answer.Valid() {
		return fmt.Errorf("answer harus 'will_renew', 'will_not', atau 'unsure'")
	}
	// `pending` adalah keadaan AWAL, bukan jawaban. Menerimanya sebagai jawaban
	// membuat MC bisa "menjawab" dengan mengosongkan kembali, dan ST kehilangan
	// perbedaan antara belum dijawab dan sengaja dikosongkan.
	if in.Answer == RenewalPending {
		return fmt.Errorf("answer tidak boleh 'pending' — itu keadaan awal, bukan jawaban")
	}
	return nil
}

// RenewalFilter menyaring daftar permintaan.
type RenewalFilter struct {
	ChapterID string
	Answer    string
	Period    string
	Limit     int
	Offset    int
}
