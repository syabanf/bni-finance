package domain

import "testing"

func TestCollectionRate(t *testing.T) {
	kasus := []struct {
		diterima, ditagih int64
		ingin             int
	}{
		{0, 0, 0}, // tanpa tagihan, bukan pembagian dengan nol
		{500, 0, 0},
		{1, 2, 50},
		{1, 3, 33},
		{2, 3, 67},
		{525, 1000, 53}, // setengah dibulatkan ke atas
		{5, 1000, 1},
		{4, 1000, 0},
		{3_000_000, 2_000_000, 150}, // penerimaan periode ini boleh melebihi tagihannya
	}
	for _, k := range kasus {
		if got := CollectionRate(k.diterima, k.ditagih); got != k.ingin {
			t.Errorf("CollectionRate(%d, %d) = %d, ingin %d", k.diterima, k.ditagih, got, k.ingin)
		}
	}
}
