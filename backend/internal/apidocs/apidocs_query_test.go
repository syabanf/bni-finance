package apidocs

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Guard untuk QUERY PARAMETER.
//
// Guard yang sudah ada membandingkan RUTE terhadap spesifikasi, dan itu memang
// menangkap endpoint yang lupa didokumentasikan. Tapi ia buta terhadap apa yang
// dibaca sebuah endpoint dari query string — dan kebutaan itu nyata, bukan
// hipotesis: `?aging=` dan `?summary=` ditambahkan ke handler invoice, seluruh
// guard tetap hijau, dan deskripsi `q` di spesifikasi masih menyatakan pencarian
// hanya mencakup nomor invoice padahal sudah mencakup nama member juga.
//
// Dokumentasi yang salah lebih buruk daripada yang tidak ada. Yang tidak ada
// membuat orang membaca kode; yang salah membuat mereka percaya dan berhenti.
//
// ARAHNYA SENGAJA SATU. Yang dijaga: setiap parameter yang DIBACA KODE harus
// terdokumentasi. Arah sebaliknya — parameter yang tertulis di spesifikasi tapi
// tak terbaca kode — tidak dijaga di sini, karena parameter bisa saja dibaca
// lewat middleware atau helper yang tidak terlihat dari badan handler, dan
// kegagalan palsu akan membuat guard ini dimatikan orang.

// bacaanQuery mengumpulkan nama parameter dari httpx.Query(r, "x") dan
// httpx.QueryInt(r, "x", ...) di dalam sebuah badan fungsi.
func bacaanQuery(fn *ast.FuncDecl) []string {
	var out []string
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "httpx" {
			return true
		}
		if sel.Sel.Name != "Query" && sel.Sel.Name != "QueryInt" {
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if nama, err := strconv.Unquote(lit.Value); err == nil {
			out = append(out, nama)
		}
		return true
	})
	return out
}

// namaHandler membuka bungkus seperti auth.RequireAdmin(h.create) sampai
// menemukan selector terdalam, lalu mengembalikan nama metodenya.
func namaHandler(e ast.Expr) string {
	for {
		switch v := e.(type) {
		case *ast.CallExpr:
			if len(v.Args) == 0 {
				return ""
			}
			e = v.Args[len(v.Args)-1]
		case *ast.SelectorExpr:
			return v.Sel.Name
		default:
			return ""
		}
	}
}

// queryPerOperasi memetakan "METHOD /path" ke parameter yang dibaca handlernya.
func queryPerOperasi(t *testing.T) map[string][]string {
	t.Helper()

	out := map[string][]string{}
	root := ".."
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		berkas, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil // Berkas yang tidak terurai bukan urusan guard ini.
		}

		// Kumpulkan seluruh deklarasi fungsi di berkas ini agar handler bisa
		// dicari berdasarkan nama.
		fungsi := map[string]*ast.FuncDecl{}
		for _, d := range berkas.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				fungsi[fd.Name.Name] = fd
			}
		}

		ast.Inspect(berkas, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "HandleFunc" {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			pola, err := strconv.Unquote(lit.Value)
			if err != nil || !routePattern.MatchString(pola) {
				return true
			}
			fd := fungsi[namaHandler(call.Args[1])]
			if fd == nil {
				return true
			}
			if p := bacaanQuery(fd); len(p) > 0 {
				out[pola] = append(out[pola], p...)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("baca sumber: %v", err)
	}
	return out
}

type paramSpec struct {
	Name string `json:"name"`
	In   string `json:"in"`
	Ref  string `json:"$ref"`
}

type operasiParam struct {
	Parameters []paramSpec `json:"parameters"`
}

// komponenParam membaca components.parameters agar $ref bisa ditelusuri.
//
// Versi pertama guard ini membaca name/in inline saja, lalu melaporkan delapan
// parameter "tidak terdokumentasi" yang sebenarnya terdokumentasi rapi lewat
// $ref ke komponen bersama. Guard yang menuduh secara keliru akan dimatikan
// orang, dan setelah itu ia tidak menjaga apa pun.
func komponenParam(t *testing.T) map[string]paramSpec {
	t.Helper()
	var s struct {
		Components struct {
			Parameters map[string]paramSpec `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(SpecJSON(), &s); err != nil {
		t.Fatalf("openapi.json tidak valid: %v", err)
	}
	return s.Components.Parameters
}

func TestSetiapQueryParamTerdokumentasi(t *testing.T) {
	s := loadSpec(t)
	komponen := komponenParam(t)
	terbaca := queryPerOperasi(t)

	// Pemindai yang rusak diam-diam akan meloloskan segalanya. Handler invoice
	// saja membaca sebelas parameter, jadi angka jauh di bawah itu berarti
	// pemindainya yang patah, bukan kodenya yang bersih.
	if len(terbaca) < 3 {
		t.Fatalf("hanya menemukan %d operasi berparameter — pemindai kemungkinan rusak", len(terbaca))
	}

	var kurang []string
	for pola, dipakai := range terbaca {
		bagian := strings.SplitN(pola, " ", 2)
		metode, jalur := strings.ToLower(bagian[0]), bagian[1]

		item, ada := s.Paths[jalur]
		if !ada {
			continue // Sudah jadi urusan TestEveryRouteIsDocumented.
		}
		var op operasiParam
		if mentah, ada := item[metode]; ada {
			_ = json.Unmarshal(mentah, &op)
		}
		didokumentasikan := map[string]bool{}
		for _, p := range op.Parameters {
			if p.Ref != "" {
				p = komponen[strings.TrimPrefix(p.Ref, "#/components/parameters/")]
			}
			if p.In == "query" {
				didokumentasikan[p.Name] = true
			}
		}
		for _, nama := range dipakai {
			if !didokumentasikan[nama] {
				kurang = append(kurang, pola+" ?"+nama)
			}
		}
	}

	sort.Strings(kurang)
	if len(kurang) > 0 {
		t.Errorf("parameter dibaca kode tapi tidak ada di openapi.yaml:\n  %s\n\n"+
			"Dokumentasikan, lalu jalankan `make docs`.", strings.Join(kurang, "\n  "))
	}
}
