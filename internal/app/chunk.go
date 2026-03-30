package app

type ChunkType string

const (
	ChunkTypeFunction  ChunkType = "FUNCTION"
	ChunkTypeStruct    ChunkType = "STRUCT"
	ChunkTypeInterface ChunkType = "INTERFACE"
	ChunkTypeFile      ChunkType = "FILE"
)

type Chunk struct {
	Id           string    `json:"id" db:"id"`
	Type         ChunkType `json:"type" db:"type"`
	Name         string    `json:"name" db:"name"`
	FilePath     string    `json:"file_path" db:"file_path"`
	Content      string    `json:"content" db:"content"`
	StartLine    int       `json:"start_line" db:"start_line"`
	EndLine      int       `json:"end_line" db:"end_line"`
	Dependencies []string  `json:"dependencies" db:"dependencies"` // List of Chunk IDs this chunk depends on
}
