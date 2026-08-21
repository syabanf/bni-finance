package paperid

import "strings"

// normalizePhone mengubah nomor Indonesia ke format internasional tanpa '+',
// yang dipakai Paper.id untuk mengantar WhatsApp.
//
// Data member disimpan dalam format lokal — 082240274833 — karena begitulah
// orang menuliskannya. Paper.id mengantar WhatsApp ke nomor apa adanya, jadi
// mengirim angka berawalan 0 berarti pesannya tidak sampai ke siapa pun.
//
// Frontend sudah melakukan konversi ini sejak awal untuk tautan wa.me; backend
// tidak, sehingga setiap invoice yang dikirim dengan kanal WhatsApp menyala
// melaporkan sukses tanpa satu pesan pun tiba. Kegagalan diam-diam, dan justru
// pada kanal yang paling dipakai.
//
// Nomor yang sudah berformat 62 dibiarkan; yang kosong dikembalikan kosong
// supaya pemeriksaan "member belum punya nomor telepon" di atasnya tetap yang
// memutuskan, bukan fungsi ini.
func normalizePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch {
	case d == "":
		return ""
	case strings.HasPrefix(d, "0"):
		return "62" + strings.TrimPrefix(d, "0")
	case strings.HasPrefix(d, "62"):
		return d
	default:
		return "62" + d
	}
}
