package apidocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Nama berkas tes harus menamai berkas sumber yang diujinya.
//
//	<sumber>_test.go            phone.go        -> phone_test.go
//	<sumber>_<aspek>_test.go    repository.go   -> repository_live_test.go
//
// Gunanya sederhana: dari daftar berkas saja orang bisa tahu apa yang diuji dan
// di mana kodenya. Nama yang berdiri sendiri — sendrecord_test.go,
// statusof_test.go, persist_test.go — memaksa membuka berkasnya dulu untuk tahu
// paket mana yang sebenarnya sedang diuji, dan seiring waktu berkas tes berhenti
// punya hubungan yang bisa ditelusuri dengan kode yang dijaganya.
//
// Tes ini hidup di apidocs karena paket ini memang tempat penjaga lintas-repo
// lainnya berada.
func TestNamaBerkasTesMenamaiSumbernya(t *testing.T) {
	// internal/api dikecualikan dengan sengaja: paket itu harness integrasi
	// yang menguji API sebagai KOTAK HITAM. Satu-satunya sumbernya router.go,
	// sementara tesnya berbicara tentang autentikasi, konkurensi, beban, dan
	// perjalanan end-to-end. Memaksakan awalan "router_" pada semuanya akan
	// menghasilkan nama yang lebih buruk, bukan lebih baik.
	exempt := map[string]bool{"api": true}

	root := filepath.Join("..", "..", "internal")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("baca internal/: %v", err)
	}

	for _, dir := range entries {
		if !dir.IsDir() || exempt[dir.Name()] {
			continue
		}
		pkg := filepath.Join(root, dir.Name())
		files, err := os.ReadDir(pkg)
		if err != nil {
			t.Fatalf("baca %s: %v", pkg, err)
		}

		sources := map[string]bool{}
		var tests []string
		for _, f := range files {
			n := f.Name()
			if !strings.HasSuffix(n, ".go") {
				continue
			}
			if strings.HasSuffix(n, "_test.go") {
				tests = append(tests, n)
				continue
			}
			sources[strings.TrimSuffix(n, ".go")] = true
		}

		for _, tf := range tests {
			prefix := strings.TrimSuffix(tf, "_test.go")
			if sources[prefix] {
				continue
			}
			matched := false
			for s := range sources {
				if strings.HasPrefix(prefix, s+"_") {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("internal/%s/%s: awalannya %q tidak menamai berkas sumber mana pun di paket ini.\n"+
					"    sumber yang ada: %s\n"+
					"    pakai <sumber>_test.go atau <sumber>_<aspek>_test.go",
					dir.Name(), tf, prefix, strings.Join(keys(sources), ", "))
			}
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
