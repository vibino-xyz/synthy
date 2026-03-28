package app

// TODO
// Instead of Caller and Callee, we can have CallerID and CalleeID
// which will act as foreign keys to the Symbol table
type CallEdge struct {
	Id     string `json:"id" db:"id"`
	Caller string `json:"caller" db:"caller"`
	Callee string `json:"callee" db:"callee"`
}
