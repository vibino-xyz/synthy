package rpc

import (
	"context"

	contracts "github.com/vibino-xyz/protos/contracts/build"
	"github.com/vibino-xyz/synthy/internal/usecase/retrieval"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetrievalServer adapts synthy's retrieval pipeline to the RetrievalService
// gRPC contract. compass (and any future query engine) calls Retrieve to obtain
// assembled code context for a natural-language query, then runs its own LLM.
type RetrievalServer struct {
	contracts.UnimplementedRetrievalServiceServer
	pipeline *retrieval.RetrievalPipeline
}

func NewRetrievalServer(pipeline *retrieval.RetrievalPipeline) *RetrievalServer {
	return &RetrievalServer{pipeline: pipeline}
}

func (s *RetrievalServer) Retrieve(ctx context.Context, req *contracts.RetrieveRequest) (*contracts.RetrieveResponse, error) {
	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	// organization_id is accepted for forward compatibility (per-tenant index
	// scoping) but not yet used — the pipeline queries a shared namespace that
	// matches the ingestion side.
	rctx, err := s.pipeline.Retrieve(ctx, req.GetQuery(), int(req.GetTopK()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "retrieve: %v", err)
	}

	chunks := make([]*contracts.RetrievedChunk, 0, len(rctx.Chunks))
	for _, c := range rctx.Chunks {
		var summary string
		if c.Summary != nil {
			summary = *c.Summary
		}
		chunks = append(chunks, &contracts.RetrievedChunk{
			Id:           c.ID,
			RepositoryId: c.RepositoryID,
			FileId:       c.FileID,
			StartLine:    int32(c.StartLine),
			EndLine:      int32(c.EndLine),
			Type:         string(c.Type),
			Language:     string(c.Language),
			Summary:      summary,
		})
	}

	return &contracts.RetrieveResponse{
		FormattedContext: rctx.Formatted,
		Chunks:           chunks,
	}, nil
}
