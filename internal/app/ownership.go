package app

type OwnershipEdge struct {
	Id    string `json:"id" db:"id"`
	Owner string `json:"owner" db:"owner"`
	Child string `json:"child" db:"child"`
}
