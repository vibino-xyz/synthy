package app

type ImportEdge struct {
	Id       string `json:"id" db:"id"`
	Importer string `json:"importer" db:"importer"`
	Importee string `json:"importee" db:"importee"`
}
