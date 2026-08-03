package api_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// Tes beban WAJIB memakai loadClient, bukan http.DefaultClient atau
// &http.Client{} polos.
//
// Keduanya memakai http.DefaultTransport dengan MaxIdleConnsPerHost = 2. Di
// bawah puluhan worker paralel, koneksi tidak dipakai ulang melainkan ditutup
// dan mendekam di TIME_WAIT selama 2×MSL (30 detik di macOS). Terukur: satu
// run TestStressMixedWorkload meninggalkan 5.703 soket TIME_WAIT, sementara
// rentang port ephemeral hanya 16.384.
//
// Tiga run berurutan menghabiskan rentang itu, dan yang gagal bukan cuma tes
// ini — pgx pun tidak bisa membuka koneksi keluar, sehingga paket sync dan
// paperid ikut gagal berbarengan. Kegagalan itu berbulan-bulan tampak seperti
// flake acak karena bergantung pada berapa banyak port yang masih TIME_WAIT
// dari run sebelumnya, dan tidak pernah muncul saat satu paket dijalankan
// sendirian.
//
// Regresinya tidak akan terlihat sebagai tes merah — ia terlihat sebagai tes
// LAIN yang merah, di paket lain, secara acak. Karena itu dijaga di sumber.
func TestLoadTestsReuseConnections(t *testing.T) {
	// Hanya berkas yang benar-benar menembak request paralel bervolume tinggi.
	// Tes handler biasa yang mengirim beberapa request boleh pakai apa saja.
	files := []string{"stress_test.go", "concurrency_test.go", "e2e_test.go"}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(".", name))
			if err != nil {
				t.Fatalf("baca %s: %v", name, err)
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, name, src, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}

			ast.Inspect(f, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.SelectorExpr:
					// http.DefaultClient di mana pun.
					if id, ok := v.X.(*ast.Ident); ok &&
						id.Name == "http" && v.Sel.Name == "DefaultClient" {
						t.Errorf("%s: http.DefaultClient dipakai — "+
							"pakai loadClient(t, workers, timeout); "+
							"DefaultTransport hanya menyimpan 2 idle conn per host "+
							"dan akan menghabiskan port ephemeral mesin",
							fset.Position(v.Pos()))
					}
				case *ast.CompositeLit:
					// &http.Client{...} tanpa Transport eksplisit.
					sel, ok := v.Type.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					id, ok := sel.X.(*ast.Ident)
					if !ok || id.Name != "http" || sel.Sel.Name != "Client" {
						return true
					}
					for _, el := range v.Elts {
						kv, ok := el.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Transport" {
							return true // Transport disetel sendiri — itu boleh.
						}
					}
					t.Errorf("%s: &http.Client{} tanpa Transport — "+
						"pakai loadClient(t, workers, timeout)",
						fset.Position(v.Pos()))
				}
				return true
			})
		})
	}
}

// loadClient harus benar-benar menyetel batas idle conn sesuai jumlah worker.
// Tanpa ini, seseorang bisa "memperbaiki" tes di atas dengan membungkus
// Transport kosong — yang lolos pemeriksaan sumber tetapi tetap MaxIdleConns
// PerHost = 2, dan bug-nya kembali persis seperti semula.
func TestLoadClientLimitsAreReal(t *testing.T) {
	const workers = 64
	c := loadClient(t, workers, 0)

	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("loadClient harus memakai *http.Transport, dapat %T", c.Transport)
	}
	if tr.MaxIdleConnsPerHost < workers {
		t.Errorf("MaxIdleConnsPerHost = %d, harus >= %d worker — "+
			"selisihnya jadi soket TIME_WAIT", tr.MaxIdleConnsPerHost, workers)
	}
	if tr.MaxIdleConns < workers {
		t.Errorf("MaxIdleConns = %d, harus >= %d", tr.MaxIdleConns, workers)
	}
}
