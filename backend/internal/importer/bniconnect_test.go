package importer

import (
	"context"
	"strings"
	"testing"
)

// Membangun SpreadsheetML tiruan dengan tata letak laporan BNI Connect: sel
// yang melompat lewat ss:Index, judul laporan di baris pertama, dan beberapa
// tabel bertumpuk. Data nyata dari laporan sungguhan tidak dipakai di sini:
// repositori ini publik, dan nama serta nomor telepon orang bukan fixture.
func sml(rows ...string) string {
	return `<?xml version="1.0"?><Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"><Worksheet ss:Name="Report"><Table>` +
		strings.Join(rows, "") + `</Table></Worksheet></Workbook>`
}

var escapeXML = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func sel(index int, v string) string {
	v = escapeXML.Replace(v)
	if index > 0 {
		return `<Cell ss:Index="` + itoa(index) + `"><Data ss:Type="String">` + v + `</Data></Cell>`
	}
	return `<Cell><Data ss:Type="String">` + v + `</Data></Cell>`
}

func baris(cells ...string) string { return "<Row>" + strings.Join(cells, "") + "</Row>" }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func stubBNI() *stub {
	return &stub{
		chapters: map[string]ChapterRow{"ch-rise": {ID: "ch-rise", Name: "Rise", DisplayName: "BNI Rise"}},
		members: map[string]MemberRow{
			"mem-001": {ID: "mem-001", ChapterID: "ch-rise", Name: "Budi Santoso", Status: "active",
				RenewalDate: "2026-11-01"},
			// Nama yang sama di chapter LAIN adalah orang lain.
			"mem-009": {ID: "mem-009", ChapterID: "ch-lain", Name: "Citra Dewi", Status: "active"},
		},
	}
}

// LAPORAN DUES: empat tabel di satu lembar, kolom tersebar, tanpa id.
//
// Yang dijaga: tanggal jatuh tempo dari tabel pertama sampai ke member yang
// cocok namanya; anggota di tabel "keluar" berpindah ke inactive; baris
// pemisah dan tanggal parameter tidak menjadi member; nama yang sama di
// chapter lain tidak tersentuh.
func TestBNIConnectDuesMembacaSemuaTabel(t *testing.T) {
	doc := sml(
		baris(sel(0, "Chapter ► Membership Dues Report")),
		baris(sel(0, "Chapter:"), sel(12, "Rise")),
		baris(sel(2, "Member Name"), sel(9, "Industry"), sel(12, "Type"), sel(15, "Membership Status"), sel(18, "Due Date")),
		baris(sel(0, "1"), sel(0, "Budi Santoso"), sel(9, "Konstruksi"), sel(12, "President, Member"), sel(15, "Active"), sel(18, "2027-05-01T00:00:00.000")),
		baris(sel(0, "2"), sel(0, "Citra Dewi"), sel(9, "Media"), sel(12, "Member"), sel(15, "Active"), sel(18, "2027-03-01T00:00:00.000")),
		baris(sel(0, "Late Members Since")),
		baris(sel(0, "2026-09-01T00:00:00.000")),
		baris(sel(4, "Member Name"), sel(12, "Industry"), sel(16, "Due Date"), sel(20, "Membership Status")),
		baris(sel(0, "Expired or Dropped Members Since")),
		baris(sel(0, "2026-09-01T00:00:00.000")),
		baris(sel(0, "Member Name"), sel(10, "Industry"), sel(14, "Due Date"), sel(19, "Membership Status")),
		baris(sel(0, "Dedi Keluar"), sel(10, "Apparel"), sel(14, "2026-10-01T00:00:00.000"), sel(19, "Dropped")),
	)
	st := stubBNI()
	h := jalankanOpsi(t, st, JenisMember, doc, true, Opsi{Chapter: "ch-rise"})

	if h.Format != FormatSpreadsheetML {
		t.Errorf("format = %q", h.Format)
	}
	if h.Total != 3 || h.Diperbarui != 1 || h.Baru != 2 || h.Ditolak != 0 {
		t.Fatalf("ringkasan salah: total=%d baru=%d diperbarui=%d ditolak=%d\n%+v",
			h.Total, h.Baru, h.Diperbarui, h.Ditolak, h.Baris)
	}
	byName := map[string]MemberRow{}
	for _, r := range st.tulisMem {
		byName[r.Name] = r
	}
	if r := byName["Budi Santoso"]; r.ID != "mem-001" || r.RenewalDate != "2027-05-01" || r.BusinessField != "Konstruksi" {
		t.Errorf("Budi harus diperbarui lewat id lamanya dengan jatuh tempo baru: %+v", r)
	}
	if r := byName["Citra Dewi"]; r.ID == "mem-009" || !strings.HasPrefix(r.ID, "bc-rise-") {
		t.Errorf("Citra di chapter lain tidak boleh tersentuh; yang di Rise harus member baru bc-…: %+v", r)
	}
	if r := byName["Dedi Keluar"]; r.Status != "inactive" || r.RenewalDate != "2026-10-01" {
		t.Errorf("anggota Dropped harus inactive dengan jatuh tempo terakhirnya: %+v", r)
	}
	for nama := range byName {
		if strings.HasPrefix(nama, "20") || strings.Contains(nama, "Since") {
			t.Errorf("baris pemisah %q menjadi member", nama)
		}
	}
}

// LAPORAN ROSTER: judul dua baris, klasifikasi bertingkat, tanpa status.
func TestBNIConnectRosterTidakMengubahStatus(t *testing.T) {
	doc := sml(
		baris(sel(0, "Member Name"), sel(3, "Classification"), sel(4, "Company Name"), sel(5, "Phone"), sel(6, "Data shows last 90 days")),
		baris(sel(6, "G"), sel(7, "R"), sel(8, "V")),
		baris(sel(0, "Budi Santoso"), sel(3, "Construction > Window & Glass > Window & Glass"), sel(4, "Hebrew Glass"), sel(5, "+628111"), sel(6, "17")),
	)
	st := stubBNI()
	st.members["mem-001"] = MemberRow{ID: "mem-001", ChapterID: "ch-rise", Name: "Budi Santoso", Status: "visitor"}
	h := jalankanOpsi(t, st, JenisMember, doc, true, Opsi{Chapter: "ch-rise"})
	if h.Total != 1 || h.Diperbarui != 1 {
		t.Fatalf("ringkasan salah: %+v", h.Baris)
	}
	r := st.tulisMem[0]
	if r.Status != "visitor" {
		t.Errorf("Roster tidak punya kolom status; status tersimpan %q tidak boleh berubah, dapat %q", "visitor", r.Status)
	}
	if r.BusinessField != "Window & Glass" || r.Company != "Hebrew Glass" || r.Phone != "+628111" {
		t.Errorf("kolom roster tidak terbaca: %+v", r)
	}
}

// LAPORAN LENGTH: nama dari dua kolom, tanggal bergabung dari Cumulative Start.
func TestBNIConnectLengthMengisiTanggalBergabung(t *testing.T) {
	doc := sml(
		baris(sel(0, "Chapter Name"), sel(0, "First Name"), sel(0, "Last Name"), sel(0, "Cumulative Length"), sel(0, "Cumulative Start Date"), sel(0, "Recent Length"), sel(0, "Recent Start date")),
		baris(sel(0, "Rise"), sel(0, "Budi"), sel(0, "Santoso"), sel(0, "3 years (0 months)"), sel(0, "2023-10-01T00:00:00.000"), sel(0, "3 years (0 months)"), sel(0, "2023-10-01T00:00:00.000")),
	)
	st := stubBNI()
	h := jalankanOpsi(t, st, JenisMember, doc, true, Opsi{Chapter: "ch-rise"})
	if h.Total != 1 || h.Diperbarui != 1 {
		t.Fatalf("ringkasan salah: %+v", h.Baris)
	}
	if r := st.tulisMem[0]; r.ID != "mem-001" || r.JoinedDate != "2023-10-01" {
		t.Errorf("Budi harus cocok lewat 'First Last' dan dapat tanggal bergabung: %+v", r)
	}
	ada := false
	for _, p := range h.Peringatan {
		if strings.Contains(p, "Recent") {
			ada = true
		}
	}
	if !ada {
		t.Error("peringatan bahwa kolom Recent tidak disimpan harus muncul")
	}
}

// Tanpa chapter tujuan, laporan BNI Connect tidak boleh ditebak-tebak.
func TestBNIConnectMenuntutChapterTujuan(t *testing.T) {
	doc := sml(baris(sel(0, "Member Name"), sel(3, "Classification"), sel(4, "Company Name"), sel(5, "Phone")))
	_, err := NewService(stubBNI()).Jalankan(context.Background(), JenisMember, []byte(doc), false, Opsi{})
	if err == nil || !strings.Contains(err.Error(), "chapter tujuan") {
		t.Fatalf("harus menolak dengan menyebut chapter tujuan, dapat %v", err)
	}
}

// Berkas CSV biasa dengan kolom id TIDAK boleh tersesat ke jalur BNI Connect.
func TestBerkasBiasaTidakDikiraBNIConnect(t *testing.T) {
	h := jalankanOpsi(t, stubBNI(), JenisMember,
		"id,name,chapter_id\nmem-001,Budi Santoso,ch-rise\n", false, Opsi{})
	if h.Format != FormatCSV || len(h.Peringatan) != 0 {
		t.Errorf("berkas biasa harus lewat jalur generik tanpa peringatan BNI Connect: %+v", h)
	}
}
