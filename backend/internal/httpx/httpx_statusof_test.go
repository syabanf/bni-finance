package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// StatusOf dan Fail harus selalu sepakat. Keduanya dipisah supaya perekam
// blackbox bisa tahu status tanpa menulis respons — dan begitu keduanya
// menyimpang, catatan diagnostik mulai berbohong tentang apa yang diterima
// klien. Tes ini menjalankan Fail sungguhan lalu membandingkan kode yang
// benar-benar tertulis dengan yang dilaporkan StatusOf.
func TestStatusOfSepakatDenganFail(t *testing.T) {
	kasus := []struct {
		nama string
		err  error
	}{
		{"httpx.Error 409", Conflict("bentrok")},
		{"httpx.Error 503", NewError(http.StatusServiceUnavailable, "mati", nil)},
		{"sentinel not found", ErrNotFound},
		{"not found terbungkus", fmt.Errorf("baca baris: %w", ErrNotFound)},
		{"galat biasa", errors.New("apa pun")},
		{"uuid rusak", errors.New(`ERROR: invalid input syntax for type uuid (SQLSTATE 22P02)`)},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			w := httptest.NewRecorder()
			Fail(w, k.err)
			if got := StatusOf(k.err); got != w.Code {
				t.Errorf("StatusOf = %d tetapi Fail menulis %d", got, w.Code)
			}
			var body map[string]any
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Errorf("body bukan JSON: %v", err)
			}
		})
	}
}

func TestStatusOfNilAdalah200(t *testing.T) {
	if got := StatusOf(nil); got != http.StatusOK {
		t.Errorf("StatusOf(nil) = %d, mau 200", got)
	}
}
