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

// known memetakan field yang kita pahami, per objek. Disusun dari dokumentasi
// resmi Paper.id, mencakup ketiga keluarga callback sekaligus.
var known = map[string]map[string]bool{
	"": {
		// Payment Callback (Pembayaran Masuk / Keluar)
		"ref_id": true, "external_id": true, "message": true,
		"payment_date": true, "payment_info": true, "additional_info": true,
		// Invoice Callback — dokumentasi memakai data.invoice, dashboard
		// menaruh invoice langsung di akar. Keduanya sah.
		"data": true, "invoice": true,
		// Reconciliation Callback
		"reconciled_amount": true, "reconciliation_date": true, "source": true,
		// Paylater Diajukan
		"transaction_id": true, "transaction_date": true, "transaction_status": true,
		// Pembayaran ke Supplier — seluruh detailnya datar di akar
		"company_id": true, "company_name": true, "partner_id": true,
		"partner_name": true, "payment_id": true, "disbursement_id": true,
		"status": true, "account_holder_name": true, "account_number": true,
		"account_type": true, "amount": true, "bank_code": true,
		"currency": true, "completed_time": true,
	},
	"payment_info": {
		"channel": true, "method": true, "status": true, "message": true,
		"source": true, "event": true, "payment_type": true, "additional_info": true,
	},
	"additional_info": {"invoices": true},
	"data": {
		// Invoice & Sisa Tagihan
		"invoice": true, "payment": true, "connected_documents": true,
		// Static VA
		"additional_info": true, "message": true, "partner": true,
		"payment_date": true, "payment_info": true,
		// Disbursement Pay In
		"id": true, "type": true, "status": true, "datetime": true,
		"channel": true, "account_number": true, "amount": true, "fee": true,
		"received_amount": true, "currency": true, "active_balance_after": true,
		"payment_references": true,
	},
}

// keluarga menggolongkan payload sebelum diperiksa.
//
// Tanpa ini, inspektor menuntut hal yang tidak pernah ada pada bentuk yang
// benar: callback Pembayaran ke Supplier dan Disbursement memang TIDAK punya
// invoice maupun payment_info — keduanya memberi tahu uang keluar, bukan
// tagihan lunas. Mengeluhkan itu setiap kali membuat catatan berbunyi terus,
// dan catatan yang selalu berbunyi berhenti dibaca tepat saat ia penting.
func keluargaPencairan(top map[string]json.RawMessage) bool {
	if _, ada := top["disbursement_id"]; ada {
		return true
	}
	if d, ok := objectAt(top, "data"); ok {
		if _, ada := d["payment_references"]; ada {
			return true
		}
	}
	return false
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

	// Invoice Callback tidak punya payment_info sama sekali, dan itu sah —
	// detailnya ada di data.invoice. Memeriksanya seperti Payment Callback
	// akan mengeluh atas bentuk yang benar.
	_, invoiceCallback := objectAt(top, "data")
	pencairan := keluargaPencairan(top)
	if pencairan {
		// Tidak ada pelunasan yang diharapkan; cukup periksa field tak dikenal.
		if d, ok := objectAt(top, "data"); ok {
			notes = append(notes, unknownIn("data", d)...)
		}
		sort.Strings(notes)
		return notes
	}

	// Static VA membungkus payment_info di dalam data. Mencarinya hanya di akar
	// membuat seluruh callback Static VA lolos tanpa diperiksa.
	nested, ok := objectAt(top, "payment_info")
	if !ok {
		if d, has := objectAt(top, "data"); has {
			nested, ok = objectAt(d, "payment_info")
		}
	}
	if ok {
		// Objek bernama metode pembayaran — bank_transfer, qris, credit_card,
		// ewallet, static_va, dan apa pun yang Paper.id tambahkan nanti — bukan
		// field tak dikenal. Isinya justru tempat status dan nominal berada.
		metode := map[string]json.RawMessage{}
		for k, v := range nested {
			if !known["payment_info"][k] {
				var d paymentDetail
				if json.Unmarshal(v, &d) == nil {
					continue // objek metode; sah
				}
				metode[k] = v
			}
		}
		notes = append(notes, unknownIn("payment_info", metode)...)

		_, punyaTransaksi := top["transaction_status"]
		if summarizePayment(json.RawMessage(mustJSON(nested))).Status == "" && !punyaTransaksi {
			notes = append(notes, "status pembayaran TIDAK ADA — tidak di payment_info.status "+
				"maupun di objek metodenya; tanpa itu tidak ada callback yang bisa melunasi apa pun")
		}
	} else if !invoiceCallback {
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
	// Invoice Callback membawa identitasnya di data.invoice — atau, pada bentuk
	// dashboard, di invoice akar.
	if d, ok := objectAt(top, "data"); ok {
		notes = append(notes, unknownIn("data", d)...)
		if inv, ok := objectAt(d, "invoice"); ok {
			notes = append(notes, unknownInvoiceFields(inv)...)
			if _, has := inv["id"]; has {
				punyaIdentitas = true
			}
			if _, has := inv["number"]; has {
				punyaIdentitas = true
			}
		}
	}
	if inv, ok := objectAt(top, "invoice"); ok {
		notes = append(notes, unknownInvoiceFields(inv)...)
		if _, has := inv["id"]; has {
			punyaIdentitas = true
		}
		if _, has := inv["number"]; has {
			punyaIdentitas = true
		}
	}
	if _, has := top["ref_id"]; has {
		punyaIdentitas = true
	}
	if _, has := top["external_id"]; has {
		punyaIdentitas = true
	}
	// Static VA menunjuk PARTNER, bukan invoice — uang masuk ke virtual account
	// miliknya dan pencocokan invoicenya menyusul lewat callback rekonsiliasi.
	if d, ok := objectAt(top, "data"); ok {
		if _, has := d["partner"]; has {
			punyaIdentitas = true
		}
	}
	if _, has := top["transaction_id"]; has {
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

// invoiceFields adalah field data.invoice yang kita pahami.
var invoiceFields = map[string]bool{
	"id": true, "number": true, "partner_id": true, "status": true,
	"amount": true, "amount_due": true, "total_amount": true, "updated_at": true,
}

func unknownInvoiceFields(m map[string]json.RawMessage) []string {
	var out []string
	for k := range m {
		if !invoiceFields[k] {
			out = append(out, "field tidak dikenal: invoice."+k)
		}
	}
	return out
}

func mustJSON(m map[string]json.RawMessage) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}
