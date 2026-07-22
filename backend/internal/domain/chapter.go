package domain

import (
	"fmt"
	"time"
)

// Chapter mirrors the `chapters` table. It originates as a read-only mirror of
// BNI Visitor Management, but is writable here so a chapter can be corrected or
// added without waiting for a sync run.
type Chapter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	AreaName    *string   `json:"areaName,omitempty"`
	CityName    *string   `json:"cityName,omitempty"`
	SyncedAt    time.Time `json:"syncedAt"`
}

// CreateChapterInput is the POST body. ID may be omitted — the database then
// generates one, which is what you want for chapters created outside BNI VM.
type CreateChapterInput struct {
	ID          *string `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName"`
	AreaName    *string `json:"areaName"`
	CityName    *string `json:"cityName"`
}

func (in CreateChapterInput) Validate() error {
	if in.Name == "" {
		return fmt.Errorf("name wajib diisi")
	}
	return nil
}

type UpdateChapterInput struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"displayName"`
	AreaName    *string `json:"areaName"`
	CityName    *string `json:"cityName"`
}

func (in UpdateChapterInput) Validate() error {
	if in.Name != nil && *in.Name == "" {
		return fmt.Errorf("name tidak boleh kosong")
	}
	if in.DisplayName != nil && *in.DisplayName == "" {
		return fmt.Errorf("displayName tidak boleh kosong")
	}
	return nil
}

// ChapterFilter drives GET /chapters.
type ChapterFilter struct {
	Search   string
	CityName string
	AreaName string
	Limit    int
	Offset   int
}
