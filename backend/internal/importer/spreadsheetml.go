package importer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
)

// Pembaca SpreadsheetML 2003, format yang dipakai ekspor BNI Connect.
//
// Berkasnya berekstensi .xls tapi isinya XML, bukan biner Excel lama maupun
// zip XLSX. Tiga laporan yang diekspor dari BNI Connect (Membership Dues,
// Chapter Roster, Membership Length) semuanya berbentuk ini, dan importer yang
// hanya mengenal CSV dan XLSX menolaknya sebagai "CSV yang rusak".
//
// Yang penting dari format ini: sel boleh melompat. <Cell ss:Index="9"> berarti
// sel itu ada di kolom ke-9 apa pun jumlah sel sebelumnya, dan laporan BNI
// Connect memakainya di mana-mana untuk membuat tata letak berjarak. Pembaca
// yang mengabaikan Index menaruh "Due Date" di kolom "Industry".

type smlWorkbook struct {
	Worksheets []struct {
		Rows []struct {
			Cells []struct {
				Index string `xml:"urn:schemas-microsoft-com:office:spreadsheet Index,attr"`
				Data  string `xml:"Data"`
			} `xml:"Cell"`
		} `xml:"Table>Row"`
	} `xml:"Worksheet"`
}

// adalahSpreadsheetML mengenali berkas dari isinya: XML yang menyebut ruang
// nama spreadsheet Microsoft di awal dokumen.
func adalahSpreadsheetML(data []byte) bool {
	kepala := data
	if len(kepala) > 2048 {
		kepala = kepala[:2048]
	}
	return bytes.Contains(kepala, []byte("<?xml")) &&
		bytes.Contains(kepala, []byte("urn:schemas-microsoft-com:office:spreadsheet"))
}

// bacaSpreadsheetML mengembalikan seluruh baris lembar pertama sebagai teks,
// dengan sel yang melompat (ss:Index) ditempatkan pada kolomnya.
func bacaSpreadsheetML(data []byte) ([]sheetRow, error) {
	var wb smlWorkbook
	if err := xml.Unmarshal(data, &wb); err != nil {
		return nil, fmt.Errorf("berkas bukan SpreadsheetML yang sah: %w", err)
	}
	if len(wb.Worksheets) == 0 {
		return nil, fmt.Errorf("berkas SpreadsheetML tidak memuat lembar apa pun")
	}
	ws := wb.Worksheets[0]
	rows := make([]sheetRow, 0, len(ws.Rows))
	for _, r := range ws.Rows {
		var row sheetRow
		for _, c := range r.Cells {
			if c.Index != "" {
				n, err := strconv.Atoi(c.Index)
				if err != nil || n < 1 {
					return nil, fmt.Errorf("ss:Index %q tidak sah", c.Index)
				}
				for len(row) < n-1 {
					row = append(row, "")
				}
			}
			row = append(row, c.Data)
		}
		rows = append(rows, row)
	}
	return rows, nil
}
