package domain

import "testing"

func TestKunciChapter(t *testing.T) {
	for in, want := range map[string]string{
		"Supreme":               "supreme",
		"BNI Supreme":           "supreme",
		"BNI Supreme Chapter":   "supreme",
		"bni  supreme-chapter ": "supreme",
		"ch-supreme":            "supreme",
		"ch-grow-chapter":       "grow",
		"Champion":              "champion",
		"Chapter":               "chapter",
		"":                      "",
	} {
		if got := KunciChapter(in); got != want {
			t.Errorf("KunciChapter(%q) = %q, harusnya %q", in, got, want)
		}
	}
}
