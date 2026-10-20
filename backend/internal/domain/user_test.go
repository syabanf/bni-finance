package domain

import "testing"

func strp(s string) *string { return &s }

// LINGKUP LEPAS DARI PERAN, KECUALI ADMIN.
//
// ST, MC, dan User masing-masing boleh nasional atau satu chapter. Admin
// ditolak bila berchapter: rute khusus admin tidak membaca lingkup, jadi
// chapter pada admin akan tampil di layar tanpa membatasi apa pun.
func TestValidateScope(t *testing.T) {
	kasus := []struct {
		role    UserRole
		chapter *string
		sah     bool
	}{
		{RoleAdmin, nil, true},
		{RoleAdmin, strp("  "), true},
		{RoleAdmin, strp("ch-a"), false},
		{RoleST, nil, true},
		{RoleST, strp("ch-a"), true},
		{RoleMC, nil, true},
		{RoleMC, strp("ch-a"), true},
		{RoleUser, nil, true},
		{RoleUser, strp("ch-a"), true},
	}
	for _, k := range kasus {
		err := User{Role: k.role, ChapterID: k.chapter}.ValidateScope()
		if (err == nil) != k.sah {
			t.Errorf("%s chapter=%v: err=%v, sah=%v", k.role, k.chapter, err, k.sah)
		}
	}
}

// ChapterScope membatasi setiap peran non-admin yang punya chapter, termasuk
// User. Admin dengan chapter (dari token lama atau baris yang lolos dari luar
// API) tetap nasional, sama seperti yang ditegakkan rute khusus admin.
func TestChapterScope(t *testing.T) {
	kasus := []struct {
		role     UserRole
		chapter  *string
		mauID    string
		terbatas bool
	}{
		{RoleST, strp("ch-a"), "ch-a", true},
		{RoleMC, strp("ch-a"), "ch-a", true},
		{RoleUser, strp("ch-a"), "ch-a", true},
		{RoleST, nil, "", false},
		{RoleUser, nil, "", false},
		{RoleST, strp("   "), "", false},
		{RoleAdmin, strp("ch-a"), "", false},
	}
	for _, k := range kasus {
		id, terbatas := AuthUser{Role: k.role, ChapterID: k.chapter}.ChapterScope()
		if id != k.mauID || terbatas != k.terbatas {
			t.Errorf("%s chapter=%v: dapat (%q,%v), mau (%q,%v)", k.role, k.chapter, id, terbatas, k.mauID, k.terbatas)
		}
	}
}

func TestRapikanChapter(t *testing.T) {
	if RapikanChapter(nil) != nil || RapikanChapter(strp("  ")) != nil {
		t.Error("kosong harus menjadi nil (nasional)")
	}
	if got := RapikanChapter(strp(" ch-a ")); got == nil || *got != "ch-a" {
		t.Errorf("dapat %v", got)
	}
}
