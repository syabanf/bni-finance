package blackbox

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	rec *Recorder
}

func NewHandler(rec *Recorder) *Handler { return &Handler{rec: rec} }

// Register wires the read/clear routes. Admin-only: recorded bodies contain
// member names, phone numbers and amounts, so this is not for every signed-in
// user.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/blackbox", auth.RequireAdmin(h.list))
	mux.HandleFunc("DELETE /api/v1/blackbox", auth.RequireAdmin(h.clear))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	entries := h.rec.List()

	// Optional filters, applied here rather than in the recorder so the buffer
	// stays a dumb ring.
	integration := httpx.Query(r, "integration")
	direction := httpx.Query(r, "direction")
	onlyFailed := httpx.Query(r, "status") == "failed"

	filtered := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if integration != "" && e.Integration != integration {
			continue
		}
		if direction != "" && e.Direction != direction {
			continue
		}
		if onlyFailed && e.Success {
			continue
		}
		filtered = append(filtered, e)
	}

	limit := httpx.QueryInt(r, "limit", 100, 1, 500)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	httpx.JSON(w, http.StatusOK, httpx.List(filtered, len(filtered), limit, 0))
}

func (h *Handler) clear(w http.ResponseWriter, r *http.Request) {
	h.rec.Clear()
	w.WriteHeader(http.StatusNoContent)
}
