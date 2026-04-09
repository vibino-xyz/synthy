package imports

import "time"

type ImportEdge struct {
	ID           string    `json:"id" db:"id"`
	RepositoryID string    `json:"repository_id" db:"repository_id"`
	FromFileID   string    `json:"from_file_id" db:"from_file_id"`
	ToFileID     string    `json:"to_file_id" db:"to_file_id"`
	ImportPath   string    `json:"import_path" db:"import_path"`
	IsExternal   bool      `json:"is_external" db:"is_external"`
	Alias        *string   `json:"alias" db:"alias"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
