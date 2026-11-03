package sync

import (
	"slices"
	"testing"
)

func TestMatch(t *testing.T) {
	link := func(v string) *string { return &v }

	cases := []struct {
		name      string
		remotes   []remoteRow
		stored    []storedRow
		local     map[string]string
		linked    int
		ambiguous []string
	}{
		{
			name:    "kunci unik di kedua sisi menaut",
			remotes: []remoteRow{{ID: "vm", Key: "budi", Label: "Budi"}},
			stored:  []storedRow{{ID: "bc", Keys: []string{"budi"}}},
			local:   map[string]string{"vm": "bc"},
			linked:  1,
		},
		{
			name:    "tautan lama menang atas kunci dan tidak dihitung",
			remotes: []remoteRow{{ID: "vm", Key: "nama baru", Label: "Baru"}},
			stored: []storedRow{
				{ID: "bc", BNIVMID: link("vm"), Keys: []string{"nama lama"}},
				{ID: "lain", Keys: []string{"nama baru"}},
			},
			local: map[string]string{"vm": "bc"},
		},
		{
			name:    "id yang sama menang atas kunci",
			remotes: []remoteRow{{ID: "vm", Key: "budi", Label: "Budi"}},
			stored: []storedRow{
				{ID: "vm", Keys: []string{"budi"}},
				{ID: "bc", Keys: []string{"budi"}},
			},
			local: map[string]string{"vm": "vm"},
		},
		{
			name: "baris yang diklaim id tidak bisa ditaut id lain",
			remotes: []remoteRow{
				{ID: "vm-2", Key: "budi", Label: "Budi 2"},
				{ID: "vm-1", Key: "budi", Label: "Budi 1"},
			},
			stored: []storedRow{{ID: "vm-1", Keys: []string{"budi"}}},
			local:  map[string]string{"vm-1": "vm-1"},
		},
		{
			name:    "baris yang sudah tertaut bukan calon",
			remotes: []remoteRow{{ID: "vm-2", Key: "budi", Label: "Budi"}},
			stored:  []storedRow{{ID: "bc", BNIVMID: link("vm-1"), Keys: []string{"budi"}}},
			local:   map[string]string{},
		},
		{
			name:      "dua baris lokal berkunci sama",
			remotes:   []remoteRow{{ID: "vm", Key: "budi", Label: "Budi"}},
			stored:    []storedRow{{ID: "a", Keys: []string{"budi"}}, {ID: "b", Keys: []string{"budi"}}},
			local:     map[string]string{},
			ambiguous: []string{"Budi"},
		},
		{
			name: "dua baris BNI VM berkunci sama",
			remotes: []remoteRow{
				{ID: "vm-2", Key: "budi", Label: "Budi 2"},
				{ID: "vm-1", Key: "budi", Label: "Budi 1"},
			},
			stored:    []storedRow{{ID: "a", Keys: []string{"budi"}}},
			local:     map[string]string{},
			ambiguous: []string{"Budi 1", "Budi 2"},
		},
		{
			name: "dua kunci berbeda menunjuk satu baris",
			remotes: []remoteRow{
				{ID: "vm-1", Key: "supreme", Label: "Supreme"},
				{ID: "vm-2", Key: "supremejkt", Label: "Supreme Jakarta"},
			},
			stored:    []storedRow{{ID: "ch", Keys: []string{"supreme", "supremejkt"}}},
			local:     map[string]string{},
			ambiguous: []string{"Supreme", "Supreme Jakarta"},
		},
		{
			name: "id ganda di snapshot dihitung sekali",
			remotes: []remoteRow{
				{ID: "vm", Key: "budi", Label: "Budi"},
				{ID: "vm", Key: "budi", Label: "Budi"},
			},
			stored: []storedRow{{ID: "bc", Keys: []string{"budi"}}},
			local:  map[string]string{"vm": "bc"},
			linked: 1,
		},
		{
			name:    "kunci kosong tidak pernah cocok",
			remotes: []remoteRow{{ID: "vm", Key: "", Label: ""}},
			stored:  []storedRow{{ID: "bc", Keys: []string{""}}},
			local:   map[string]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := match(c.remotes, c.stored)
			if len(got.Local) != len(c.local) {
				t.Errorf("Local = %v, harusnya %v", got.Local, c.local)
			}
			for vm, want := range c.local {
				if got.Local[vm] != want {
					t.Errorf("Local[%s] = %q, harusnya %q", vm, got.Local[vm], want)
				}
			}
			if got.Linked != c.linked {
				t.Errorf("Linked = %d, harusnya %d", got.Linked, c.linked)
			}
			if !slices.Equal(got.Ambiguous, c.ambiguous) {
				t.Errorf("Ambiguous = %q, harusnya %q", got.Ambiguous, c.ambiguous)
			}
		})
	}
}
