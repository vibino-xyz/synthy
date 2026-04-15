package api

import (
	"net/http"

	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type QueryController struct {
	embeddingClient    llm.EmbeddingClient
	pineconeRepository embedding.PineconeRepository
	chunkRepository    chunker.ChunkRepository
}

func NewQueryController(
	mux *http.ServeMux,
	embeddingClient llm.EmbeddingClient,
	pineconeRepository embedding.PineconeRepository,
	chunkRepository chunker.ChunkRepository,
) *QueryController {
	c := &QueryController{
		embeddingClient:    embeddingClient,
		pineconeRepository: pineconeRepository,
		chunkRepository:    chunkRepository,
	}
	c.registerRoutes(mux)
	return c
}

func (c *QueryController) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /query", c.handleQuery)
}

func (c *QueryController) handleQuery(w http.ResponseWriter, r *http.Request) {
	//TODO
}
