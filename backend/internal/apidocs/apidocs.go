// Package apidocs serves the OpenAPI specification and a browsable reference
// generated from it.
//
// Both files are embedded, so the binary carries its own documentation and the
// page works with no network access — no CDN, no external renderer. The YAML is
// the human-edited source; the JSON is generated from it by `make docs` and is
// what the page renders, since the browser can parse JSON without a library.
package apidocs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var specYAML []byte

//go:embed openapi.json
var specJSON []byte

//go:embed docs.html
var docsHTML []byte

// SpecJSON exposes the embedded specification so tests can check it against the
// routes that are actually registered.
func SpecJSON() []byte { return specJSON }

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

// Register wires the documentation routes. They are unauthenticated: the point
// is to be readable while you are still working out how to authenticate.
//
// That does publish the shape of the API. This backend is meant to run
// server-side behind a network boundary, so that is acceptable — but do not
// expose it straight to the internet and then treat the surface as secret.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", h.yaml)
	mux.HandleFunc("GET /openapi.json", h.json)
	mux.HandleFunc("GET /docs", h.page)
}

func (h *Handler) yaml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Write(specYAML)
}

func (h *Handler) json(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(specJSON)
}

func (h *Handler) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(docsHTML)
}
