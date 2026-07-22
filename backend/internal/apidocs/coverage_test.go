package apidocs

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Documentation rots the moment it stops being checked. This test compares the
// specification against the routes that are ACTUALLY registered, so adding an
// endpoint without documenting it — or documenting one that no longer exists —
// fails the build instead of quietly misleading whoever reads /docs.
//
// Routes are read from the source with go/ast rather than a live mux, because
// http.ServeMux offers no way to enumerate its patterns.

// routePattern matches the Go 1.22 method+path form: "GET /api/v1/invoices".
var routePattern = regexp.MustCompile(`^(GET|POST|PATCH|PUT|DELETE|HEAD|OPTIONS) (/\S*)$`)

// undocumented lists routes that are deliberately absent from the spec.
var undocumented = map[string]bool{
	// Registered as a prefix handler for the file server; the spec documents
	// the actual shape as /uploads/{filename}.
	"GET /uploads/": true,
}

type route struct {
	method string
	path   string
	file   string
}

func (r route) String() string { return r.method + " " + r.path }

// collectRoutes walks internal/ and pulls every route literal out of the
// mux.HandleFunc / mux.Handle calls.
func collectRoutes(t *testing.T) []route {
	t.Helper()

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve direktori: %v", err)
	}

	var routes []route
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle") {
				return true
			}
			// Only plain string literals — a computed pattern would need the
			// route table to become explicit, which is the better fix anyway.
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			if m := routePattern.FindStringSubmatch(value); m != nil {
				rel, _ := filepath.Rel(root, path)
				routes = append(routes, route{method: m[1], path: m[2], file: rel})
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("baca sumber: %v", err)
	}

	if len(routes) < 30 {
		t.Fatalf("hanya menemukan %d rute — pemindai sumber kemungkinan rusak", len(routes))
	}
	return routes
}

type spec struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

func loadSpec(t *testing.T) spec {
	t.Helper()
	var s spec
	if err := json.Unmarshal(SpecJSON(), &s); err != nil {
		t.Fatalf("openapi.json tidak valid — jalankan `make docs`: %v", err)
	}
	if len(s.Paths) == 0 {
		t.Fatal("openapi.json tidak memuat path apa pun")
	}
	return s
}

// documentedOperations flattens the spec into a "METHOD /path" set.
func documentedOperations(s spec) map[string]bool {
	methods := map[string]bool{
		"get": true, "post": true, "patch": true, "put": true,
		"delete": true, "head": true, "options": true,
	}
	out := make(map[string]bool, len(s.Paths)*2)
	for path, item := range s.Paths {
		for method := range item {
			if methods[method] {
				out[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	return out
}

// TestEveryRouteIsDocumented is the guard that keeps /docs honest.
func TestEveryRouteIsDocumented(t *testing.T) {
	documented := documentedOperations(loadSpec(t))

	var missing []string
	for _, r := range collectRoutes(t) {
		if undocumented[r.String()] {
			continue
		}
		if !documented[r.String()] {
			missing = append(missing, fmt.Sprintf("%s  (didaftarkan di %s)", r, r.file))
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("%d endpoint tidak ada di openapi.yaml:\n  %s\n\n"+
			"Tambahkan ke internal/apidocs/openapi.yaml lalu jalankan `make docs`.",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestNoPhantomEndpoints catches the opposite drift: documentation describing
// something the server does not serve, which is worse than no documentation
// because a reader has no way to tell.
func TestNoPhantomEndpoints(t *testing.T) {
	registered := make(map[string]bool)
	for _, r := range collectRoutes(t) {
		registered[r.String()] = true
	}
	// Prefix-registered file server; documented under its real shape.
	registered["GET /uploads/{filename}"] = true

	var phantom []string
	for op := range documentedOperations(loadSpec(t)) {
		if !registered[op] {
			phantom = append(phantom, op)
		}
	}

	if len(phantom) > 0 {
		sort.Strings(phantom)
		t.Errorf("%d endpoint terdokumentasi tapi tidak terdaftar di server:\n  %s",
			len(phantom), strings.Join(phantom, "\n  "))
	}
}

// TestSpecJSONMatchesYAML guards against editing the YAML and forgetting to run
// `make docs` — the page renders the JSON, so a stale JSON means stale docs.
// Go has no YAML parser in the standard library, so this compares the set of
// path keys, which is enough to catch an endpoint added on one side only.
func TestSpecJSONMatchesYAML(t *testing.T) {
	yamlPaths := make(map[string]bool)
	inPaths := false
	for _, line := range strings.Split(string(specYAML), "\n") {
		if line == "paths:" {
			inPaths = true
			continue
		}
		if inPaths && line != "" && !strings.HasPrefix(line, " ") {
			break // sudah keluar dari blok paths (mis. `components:`)
		}
		// Kunci path selalu dua spasi indentasi dan diawali garis miring.
		if inPaths && strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":") {
			yamlPaths[strings.TrimSuffix(strings.TrimSpace(line), ":")] = true
		}
	}
	if len(yamlPaths) == 0 {
		t.Fatal("tidak menemukan path apa pun di openapi.yaml — pemindai rusak")
	}

	jsonPaths := loadSpec(t).Paths
	for path := range yamlPaths {
		if _, ok := jsonPaths[path]; !ok {
			t.Errorf("%s ada di YAML tapi tidak di JSON — jalankan `make docs`", path)
		}
	}
	for path := range jsonPaths {
		if !yamlPaths[path] {
			t.Errorf("%s ada di JSON tapi tidak di YAML — JSON sudah usang", path)
		}
	}
}

// TestPublicOperationsAreExplicit makes sure a route that skips authentication
// says so. An endpoint silently reachable without a token is exactly the kind
// of thing a reader must not have to discover by trying it.
func TestPublicOperationsAreExplicit(t *testing.T) {
	var s struct {
		Paths map[string]map[string]struct {
			Security *[]map[string][]string `json:"security"`
			Summary  string                 `json:"summary"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(SpecJSON(), &s); err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	// Routes registered outside the auth middleware, from router.go.
	publicPaths := map[string]bool{
		"/healthz":                             true,
		"/openapi.yaml":                        true,
		"/openapi.json":                        true,
		"/docs":                                true,
		"/uploads/{filename}":                  true,
		"/api/v1/auth/login":                   true,
		"/api/v1/public/invoices/{id}":         true,
		"/api/v1/public/invoices/{id}/payment": true,
		"/api/v1/webhooks/xendit":              true,
	}

	for path, item := range s.Paths {
		for method, op := range item {
			isPublic := op.Security != nil && len(*op.Security) == 0
			shouldBePublic := publicPaths[path]

			if shouldBePublic && !isPublic {
				t.Errorf("%s %s dapat diakses tanpa token tapi tidak menandai `security: []`",
					strings.ToUpper(method), path)
			}
			if isPublic && !shouldBePublic {
				t.Errorf("%s %s ditandai publik di spec, padahal berada di balik autentikasi",
					strings.ToUpper(method), path)
			}
		}
	}
}
