package importer

import (
	"reflect"
	"testing"
)

const contohSML = `<?xml version="1.0" encoding="UTF-8"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet">
 <Worksheet ss:Name="Report">
  <Table>
   <Row><Cell><Data ss:Type="String">Chapter &#9658; Membership Dues Report</Data></Cell></Row>
   <Row>
    <Cell ss:Index="2"><Data ss:Type="String">Member Name</Data></Cell>
    <Cell ss:Index="9"><Data ss:Type="String">Industry</Data></Cell>
    <Cell ss:Index="18"><Data ss:Type="String">Due Date</Data></Cell>
   </Row>
   <Row>
    <Cell><Data ss:Type="Number">1</Data></Cell>
    <Cell><Data ss:Type="String">Irfan Arsandi</Data></Cell>
    <Cell ss:Index="9"><Data ss:Type="String">IT Consultants</Data></Cell>
    <Cell ss:Index="18"><Data ss:Type="DateTime">2027-05-01T00:00:00.000</Data></Cell>
   </Row>
   <Row/>
  </Table>
 </Worksheet>
 <Worksheet ss:Name="Lembar kedua"><Table><Row><Cell><Data ss:Type="String">diabaikan</Data></Cell></Row></Table></Worksheet>
</Workbook>`

// SEL YANG MELOMPAT HARUS MENDARAT DI KOLOMNYA.
//
// Laporan BNI Connect menaruh "Due Date" di kolom ke-18 lewat ss:Index, bukan
// dengan tujuh belas sel kosong di depannya. Pembaca yang mengabaikan atribut
// itu menggeser seluruh baris ke kiri, dan tanggal jatuh tempo terbaca sebagai
// bidang usaha.
func TestBacaSpreadsheetMLMenghormatiIndex(t *testing.T) {
	rows, err := bacaSpreadsheetML([]byte(contohSML))
	if err != nil {
		t.Fatalf("baca: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("baris = %d, seharusnya 4 (termasuk baris kosong)", len(rows))
	}
	judul := rows[1]
	if judul[1] != "Member Name" || judul[8] != "Industry" || judul[17] != "Due Date" {
		t.Errorf("judul tidak pada kolomnya: %v", judul)
	}
	data := rows[2]
	mau := sheetRow{"1", "Irfan Arsandi", "", "", "", "", "", "", "IT Consultants",
		"", "", "", "", "", "", "", "", "2027-05-01T00:00:00.000"}
	if !reflect.DeepEqual(data, mau) {
		t.Errorf("baris data:\n  dapat %v\n  mau   %v", data, mau)
	}
	// Hanya lembar pertama yang dibaca, seperti XLSX.
	for _, r := range rows {
		for _, sel := range r {
			if sel == "diabaikan" {
				t.Error("lembar kedua ikut terbaca")
			}
		}
	}
}

// Deteksi dari ISI, bukan dari nama berkas: ekspor BNI Connect berekstensi
// .xls padahal isinya XML.
func TestBacaMengenaliSpreadsheetML(t *testing.T) {
	rows, format, err := Baca([]byte(contohSML))
	if err != nil {
		t.Fatalf("Baca: %v", err)
	}
	if format != FormatSpreadsheetML {
		t.Errorf("format = %q, seharusnya %q", format, FormatSpreadsheetML)
	}
	if len(rows) == 0 || rows[1][1] != "Member Name" {
		t.Errorf("isi tidak terbaca lewat Baca: %v", rows)
	}
	// Teks biasa yang kebetulan menyebut XML tidak boleh dikira SpreadsheetML.
	if _, format, _ := Baca([]byte("id,name\n1,<?xml bukan workbook\n")); format != FormatCSV {
		t.Errorf("CSV yang menyebut <?xml dikira %q", format)
	}
}
