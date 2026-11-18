package httpx

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestQueryList(t *testing.T) {
	kasus := []struct {
		nama, url string
		maks      int
		ingin     []string
	}{
		{"kosong", "/x", 10, nil},
		{"satu", "/x?ids=a", 10, []string{"a"}},
		{"spasi dan entri kosong dibuang", "/x?ids=a,%20b%20,,c", 10, []string{"a", "b", "c"}},
		{"kembar dibuang", "/x?ids=a,b,a", 10, []string{"a", "b"}},
		{"dipotong di batas", "/x?ids=a,b,c,d", 2, []string{"a", "b"}},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			got := QueryList(httptest.NewRequest("GET", k.url, nil), "ids", k.maks)
			if !reflect.DeepEqual(got, k.ingin) {
				t.Errorf("QueryList = %v, ingin %v", got, k.ingin)
			}
		})
	}
}
