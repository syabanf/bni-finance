package paperid

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Bentuk payload callback Paper.id disusun dari dokumentasi, bukan dari
// tangkapan panggilan sungguhan — dan sampai callback pertama benar-benar
// datang, tidak ada yang membuktikan tebakan itu benar.
//
// Kalau ternyata berbeda, kegagalannya SUNYI: encoding/json mengabaikan field
// yang tidak dikenal dan membiarkan field yang dikenal bernilai kosong. Callback
// dijawab 200, tidak ada invoice yang dilunasi, dan tidak ada yang merah.
//
// inspectPayload membandingkan apa yang benar-benar dikirim dengan apa yang kita
// harapkan, lalu mengembalikan catatan yang bisa dibaca manusia. Catatan itu
// ikut tersimpan di blackbox, sehingga ketidakcocokan format terlihat pada
// callback PERTAMA, bukan setelah berminggu-minggu pembayaran yang tidak
// pernah tercatat.

// known memetakan field yang kita pahami, per objek.
var known = map[string]map[string]bool{
	"": {
		"ref_id": true, "external_id": true, "payment_date": true,
		"payment_info": true, "additional_info": true,
	},
	"payment_info": {
		"method": true, "channel": true, "amount": true,
		"paid_amount": true, "paid_at": true, "status": true,
	},
	"additional_info": {"invoices": true},
}

// inspectPayload melaporkan selisih antara payload nyata dan harapan kita.
//
// Dua arah, dan yang kedua justru yang paling penting:
//
//   - field TAK DIKENAL memberi tahu Paper.id mengirim sesuatu yang kita buang
//   - field YANG DIHARAPKAN TAPI HILANG memberi tahu kita membaca nama yang
//     salah — inilah yang membuat pelunasan diam-diam tidak terjadi
func inspectPayload(raw []byte) []string {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return []string{"payload bukan objek JSON: " + err.Error()}
	}

	var notes []string
	notes = append(notes, unknownIn("", top)...)

	if nested, ok := objectAt(top, "payment_info"); ok {
		notes = append(notes, unknownIn("payment_info", nested)...)
		if _, has := nested["status"]; !has {
			notes = append(notes, "payment_info.status TIDAK ADA — tanpa itu tidak ada callback yang bisa melunasi apa pun")
		}
	} else {
		notes = append(notes, "payment_info TIDAK ADA — seluruh detail pembayaran dibaca dari sini")
	}

	// Identitas invoice: tanpa salah satu dari ini, pelunasan tidak punya
	// sasaran dan callback berakhir sebagai 200 yang tidak berbuat apa-apa.
	punyaIdentitas := false
	if nested, ok := objectAt(top, "additional_info"); ok {
		notes = append(notes, unknownIn("additional_info", nested)...)
		if invs, has := nested["invoices"]; has {
			var list []map[string]json.RawMessage
			if json.Unmarshal(invs, &list) == nil && len(list) > 0 {
				punyaIdentitas = true
				for k := range list[0] {
					if k != "uuid" && k != "number" {
						notes = append(notes, fmt.Sprintf(
							"additional_info.invoices[].%s tidak dikenal", k))
					}
				}
			}
		}
	}
	if _, has := top["ref_id"]; has {
		punyaIdentitas = true
	}
	if _, has := top["external_id"]; has {
		punyaIdentitas = true
	}
	if !punyaIdentitas {
		notes = append(notes,
			"TIDAK ADA identitas invoice — ref_id, external_id, maupun additional_info.invoices semuanya kosong; "+
				"pelunasan tidak punya sasaran")
	}

	sort.Strings(notes)
	return notes
}

func objectAt(m map[string]json.RawMessage, key string) (map[string]json.RawMessage, bool) {
	raw, ok := m[key]
	if !ok {
		return nil, false
	}
	var out map[string]json.RawMessage
	if json.Unmarshal(raw, &out) != nil {
		return nil, false
	}
	return out, true
}

func unknownIn(prefix string, m map[string]json.RawMessage) []string {
	var out []string
	for k := range m {
		if known[prefix][k] {
			continue
		}
		name := k
		if prefix != "" {
			name = prefix + "." + k
		}
		out = append(out, "field tidak dikenal: "+name)
	}
	return out
}

// formatNotes merangkai catatan untuk disematkan pada rekaman blackbox.
func formatNotes(notes []string) string {
	if len(notes) == 0 {
		return ""
	}
	return "format callback: " + strings.Join(notes, "; ")
}
