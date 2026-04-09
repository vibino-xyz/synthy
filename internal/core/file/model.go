package file

import "time"

type Language string

const (
	LanguageGo    Language = "GO"
	LanguageOther Language = "OTHER"
)

type File struct {
	ID           string    `json:"id" db:"id"`
	RepositoryID string    `json:"repository_id" db:"repository_id"`
	Path         string    `json:"path" db:"path"`
	Name         string    `json:"name" db:"name"`
	Extension    string    `json:"extension" db:"extension"`
	Checksum     string    `json:"checksum" db:"checksum"`
	Language     Language  `json:"language" db:"language"`
	IsBinary     bool      `json:"is_binary" db:"is_binary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
