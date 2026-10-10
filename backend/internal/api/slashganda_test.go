package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// RapikanPath harus terpasang di rantai NewHandler, bukan hanya ada di httpx.
//
// Dibuktikan lewat login: POST ke "//api/v1/auth/login" dengan body harus
// sampai ke handler (dijawab 401 atas kata sandi yang salah, yang berarti
// body-nya terbaca), bukan 301 dari mux. Klien di sini sengaja tidak
// mengikuti redirect supaya 301-nya terlihat apa adanya bila muncul.
func TestPathGarisMiringGandaSampaiKeHandler(t *testing.T) {
	stack := newFullServer(t)
	klien := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, _ := http.NewRequest(http.MethodPost, stack.srv.URL+"//api/v1/auth/login",
		strings.NewReader(`{"email":"tidak@ada.id","password":"salah"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := klien.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusMovedPermanently {
		t.Fatal("mux menjawab 301: RapikanPath tidak terpasang di rantai, POST webhook akan kehilangan body")
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("handler login harus terjangkau dan menolak kredensial salah dengan 401, dapat %d", res.StatusCode)
	}
}
