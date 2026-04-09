package symbol

import "time"

type SymbolType string

const (
	SymbolTypeFunction  SymbolType = "FUNCTION"
	SymbolTypeType      SymbolType = "TYPE"
	SymbolTypeVariable  SymbolType = "VARIABLE"
	SymbolTypeConstant  SymbolType = "CONSTANT"
	SymbolTypePackage   SymbolType = "PACKAGE"
	SymbolTypeField     SymbolType = "FIELD"
	SymbolTypeInterface SymbolType = "INTERFACE"
	SymbolTypeImport    SymbolType = "IMPORT"
	SymbolTypeOther     SymbolType = "OTHER"
)

type ReceiverType string

const (
	ReceiverTypeValue   ReceiverType = "VALUE"
	ReceiverTypePointer ReceiverType = "POINTER"
)

type SymbolVisibility string

const (
	SymbolVisibilityPublic  SymbolVisibility = "PUBLIC"
	SymbolVisibilityPrivate SymbolVisibility = "PRIVATE"
)

type Symbol struct {
	ID             string           `json:"id" db:"id"`
	FileID         string           `json:"file_id" db:"file_id"`
	RepositoryID   string           `json:"repository_id" db:"repository_id"`
	FqName         string           `json:"fq_name" db:"fq_name"`
	Type           SymbolType       `json:"type" db:"type"`
	Language       string           `json:"language" db:"language"`
	Signature      string           `json:"signature" db:"signature"`
	ReceiverType   *ReceiverType    `json:"receiver_type" db:"receiver_type"`
	Visibility     SymbolVisibility `json:"visibility" db:"visibility"`
	ParentSymbolId *string          `json:"parent_symbol_id" db:"parent_symbol_id"`
	StartLine      int              `json:"start_line" db:"start_line"`
	EndLine        int              `json:"end_line" db:"end_line"`
	StartColumn    int              `json:"start_column" db:"start_column"`
	EndColumn      int              `json:"end_column" db:"end_column"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`
}
