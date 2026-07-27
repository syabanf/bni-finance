package blackbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecordAndList(t *testing.T) {
	r := New(10)
	r.Record(Call{
		Integration: "paper_id", Direction: Outbound,
		Method: "POST", URL: "https://open-api.stag-v2.paper.id/api/v1/store-invoice",
		Request:  []byte(`{"number":"INV-1"}`),
		Response: []byte(`{"data":{"id":"pp-1"},"status_code":201}`),
		Status:   201, Success: true, Duration: 1500 * time.Millisecond,
	})

	got := r.List()
	if len(got) != 1 {
		t.Fatalf("harus 1 entri, dapat %d", len(got))
	}
	e := got[0]
	if e.Integration != "paper_id" || !e.Success || e.Status != 201 {
		t.Errorf("entri salah: %+v", e)
	}
	if e.DurationMS != 1500 {
		t.Errorf("durasi salah: %d", e.DurationMS)
	}
	// Bodies must survive as usable JSON, not as escaped strings.
	var req map[string]any
	if err := json.Unmarshal(e.Request, &req); err != nil || req["number"] != "INV-1" {
		t.Errorf("request tidak tersimpan sebagai JSON: %s (%v)", e.Request, err)
	}
}

func TestNewestFirstAndCapped(t *testing.T) {
	r := New(3)
	for i := 1; i <= 5; i++ {
		r.Record(Call{Integration: "x", URL: fmt.Sprintf("/call/%d", i)})
	}
	got := r.List()
	if len(got) != 3 {
		t.Fatalf("buffer harus dibatasi 3, dapat %d", len(got))
	}
	// Newest first: the last recorded call leads.
	if got[0].URL != "/call/5" || got[2].URL != "/call/3" {
		t.Errorf("urutan salah: %s … %s", got[0].URL, got[2].URL)
	}
}

// A non-JSON upstream body (an HTML error page, say) must not corrupt the
// response — it gets wrapped as a JSON string.
func TestNonJSONBodyIsWrapped(t *testing.T) {
	r := New(5)
	r.Record(Call{
		Integration: "paper_id",
		Response:    []byte("<html>502 Bad Gateway</html>"),
		Status:      502,
	})
	e := r.List()[0]
	if !json.Valid(e.Response) {
		t.Fatalf("response harus JSON valid, dapat %s", e.Response)
	}
	var s string
	if err := json.Unmarshal(e.Response, &s); err != nil || !strings.Contains(s, "502") {
		t.Errorf("isi asli hilang: %s", e.Response)
	}
}

func TestErrorIsCaptured(t *testing.T) {
	r := New(5)
	r.Record(Call{Integration: "paper_id", Err: errors.New("context deadline exceeded")})
	if e := r.List()[0]; e.Error != "context deadline exceeded" || e.Success {
		t.Errorf("error tidak tercatat: %+v", e)
	}
}

// A nil recorder must be safe: integrations call Record unconditionally.
func TestNilRecorderIsNoOp(t *testing.T) {
	var r *Recorder
	r.Record(Call{Integration: "x"}) // tidak boleh panic
	if got := r.List(); len(got) != 0 {
		t.Errorf("recorder nil harus mengembalikan kosong, dapat %d", len(got))
	}
	r.Clear()
}

func TestClear(t *testing.T) {
	r := New(5)
	r.Record(Call{Integration: "x"})
	r.Clear()
	if len(r.List()) != 0 {
		t.Error("Clear harus mengosongkan buffer")
	}
}

// Integration calls run concurrently, so the ring must not race or lose entries.
func TestConcurrentRecord(t *testing.T) {
	r := New(500)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r.Record(Call{Integration: "x", URL: fmt.Sprintf("/c/%d", n)})
		}(i)
	}
	wg.Wait()
	if got := len(r.List()); got != 100 {
		t.Errorf("harus 100 entri, dapat %d", got)
	}
}
