package calls

import "time"

type CallEdge struct {
	ID             string    `json:"id" db:"id"`
	RepositoryID   string    `json:"repository_id" db:"repository_id"`
	CallerSymbolID string    `json:"caller_symbol_id" db:"caller_symbol_id"`
	CalleeSymbolID string    `json:"callee_symbol_id" db:"callee_symbol_id"`
	FileID         string    `json:"file_id" db:"file_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
