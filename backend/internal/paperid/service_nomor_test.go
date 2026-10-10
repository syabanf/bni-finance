package paperid

import "testing"

// nomorKanonik harus membalikkan reminderNumber persis, dan tidak menyentuh
// nomor yang kebetulan mirip.
//
// "-R" diikuti angka adalah satu-satunya bentuk yang dibuat reminderNumber.
// Nomor lain yang mengandung "-R" tanpa angka, atau angka tanpa "-R", harus
// kembali apa adanya: memotong terlalu rakus bisa membuat callback atas satu
// invoice melunasi invoice lain yang nomornya menjadi sama setelah dipotong.
func TestNomorKanonik(t *testing.T) {
	kasus := []struct{ masuk, mau string }{
		{"INV-2026-023-R1", "INV-2026-023"},
		{"INV-2026-023-R12", "INV-2026-023"},
		{"INV-2026-023", "INV-2026-023"},
		{"INV-2026-023-R", "INV-2026-023-R"},
		{"INV-2026-023-RX", "INV-2026-023-RX"},
		{"INV-2026-023-R1x", "INV-2026-023-R1x"},
		{"", ""},
	}
	for _, k := range kasus {
		if got := nomorKanonik(k.masuk); got != k.mau {
			t.Errorf("nomorKanonik(%q) = %q, seharusnya %q", k.masuk, got, k.mau)
		}
	}

	// Bolak-balik: apa pun yang dibuat reminderNumber harus kembali ke asalnya.
	for n := 1; n <= 12; n++ {
		asal := "INV-2026-023"
		if got := nomorKanonik(reminderNumber(asal, n)); got != asal {
			t.Errorf("putaran ke-%d: dapat %q, seharusnya %q", n, got, asal)
		}
	}
}
