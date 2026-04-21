package api

import (
	"net/http"

	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"github.com/vibino-xyz/synthy/internal/core/llm"
	"github.com/vibino-xyz/synthy/internal/usecase/retrieval"
)

type QueryController struct {
	embeddingClient    llm.EmbeddingClient
	pineconeRepository embedding.PineconeRepository
	chunkRepository    chunker.ChunkRepository
	retrievalPipeline  *retrieval.RetrievalPipeline
}

func NewQueryController(
	mux *http.ServeMux,
	embeddingClient llm.EmbeddingClient,
	pineconeRepository embedding.PineconeRepository,
	chunkRepository chunker.ChunkRepository,
	retrievalPipeline *retrieval.RetrievalPipeline,
) *QueryController {
	c := &QueryController{
		embeddingClient:    embeddingClient,
		pineconeRepository: pineconeRepository,
		chunkRepository:    chunkRepository,
		retrievalPipeline:  retrievalPipeline,
	}
	c.registerRoutes(mux)
	return c
}

func (c *QueryController) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/query", c.handleQuery)
}

func (c *QueryController) handleQuery(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	retrievalContext, err := c.retrievalPipeline.ProcessRetrieval(ctx, query)
	if err != nil {
		http.Error(w, "failed to process retrieval: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(retrievalContext.Formatted))
}
