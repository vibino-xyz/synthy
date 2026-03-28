package app

import "time"

type SymbolKind string

const (
	SymbolKindFunction  SymbolKind = "FUNCTION"
	SymbolKindMethod    SymbolKind = "METHOD"
	SymbolKindClass     SymbolKind = "CLASS"
	SymbolKindStruct    SymbolKind = "STRUCT"
	SymbolKindInterface SymbolKind = "INTERFACE"
	SymbolKindEnum      SymbolKind = "ENUM"
	SymbolKindType      SymbolKind = "TYPE"
)

type Symbol struct {
	Id               string     `json:"id" db:"id"`
	RepositoryId     string     `json:"repositoryId" db:"repository_id"`
	FileId           string     `json:"fileId" db:"file_id"`
	Kind             SymbolKind `json:"kind" db:"kind"`
	Name             string     `json:"name" db:"name"`
	FqName           string     `json:"fqName" db:"fq_name"`
	ReceiverType     string     `json:"receiverType,omitempty" db:"receiver_type"`
	StartLine        int        `json:"startLine" db:"start_line"`
	EndLine          int        `json:"endLine" db:"end_line"`
	CodeObjectUri    string     `json:"codeObjectUri" db:"code_object_uri"`
	AstObjectUri     string     `json:"astObjectUri" db:"ast_object_uri"`
	SummaryObjectUri string     `json:"summaryObjectUri" db:"summary_object_uri"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at"`
}
