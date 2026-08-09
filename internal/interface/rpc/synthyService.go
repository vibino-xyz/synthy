package rpc

import (
	"context"

	synthyV1 "github.com/vibino-xyz/protos/api/build/synthy/v1"
	"github.com/vibino-xyz/synthy/internal/usecase/retrieval"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vibino-xyz/commons/vgrpc"
)

type SynthyService struct {
	synthyV1.UnsafeSynthyServiceServer
	pipeline *retrieval.RetrievalPipeline
}

type SynthyServiceGrpcServer struct {
	*grpc.Server
}

func NewSynthyServiceGrpcServer(pipeline *retrieval.RetrievalPipeline) *SynthyServiceGrpcServer {
	s := vgrpc.NewServer()

	synthyV1.RegisterSynthyServiceServer(s, &SynthyService{
		pipeline: pipeline,
	})

	return &SynthyServiceGrpcServer{
		Server: s,
	}
}

func (s SynthyService) Retrieve(ctx context.Context, req *synthyV1.RetrieveRequest) (*synthyV1.RetrieveResponse, error) {
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

	chunks := make([]*synthyV1.RetrievedChunk, 0, len(rctx.Chunks))
	for _, c := range rctx.Chunks {
		var summary string
		if c.Summary != nil {
			summary = *c.Summary
		}
		chunks = append(chunks, &synthyV1.RetrievedChunk{
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

	return &synthyV1.RetrieveResponse{
		FormattedContext: rctx.Formatted,
		Chunks:           chunks,
	}, nil
}
