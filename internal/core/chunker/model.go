package chunker

import "time"

type ChunkType string

const (
	ChunkTypeFunction ChunkType = "FUNCTION"
	ChunkTypeType     ChunkType = "TYPE"
	ChunkTypeBlock    ChunkType = "BLOCK"
	ChunkTypeOther    ChunkType = "OTHER"
)

type Language string

const (
	LanguageGo    Language = "GO"
	LanguageOther Language = "OTHER"
)

type CodeChunk struct {
	ID           string    `json:"id" db:"id"`
	RepositoryID string    `json:"repository_id" db:"repository_id"`
	FileID       string    `json:"file_id" db:"file_id"`
	SymbolID     string    `json:"symbol_id" db:"symbol_id"`
	Content      string    `json:"content" db:"content"`
	ContentHash  string    `json:"content_hash" db:"content_hash"`
	Type         ChunkType `json:"type" db:"type"`
	Language     Language  `json:"language" db:"language"`
	StartLine    int       `json:"start_line" db:"start_line"`
	EndLine      int       `json:"end_line" db:"end_line"`
	EmbeddingID  *string   `json:"embedding_id" db:"embedding_id"`
	Summary      *string   `json:"summary" db:"summary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
