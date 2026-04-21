package api

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Answer string `json:"answer"`
}
