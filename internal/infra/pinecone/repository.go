package pinecone

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"google.golang.org/protobuf/types/known/structpb"
)

type pineconeRepository struct {
	idxConnection *pinecone.IndexConnection
}

func NewPineconeRepository(idxConnection *pinecone.IndexConnection) embedding.PineconeRepository {
	return &pineconeRepository{idxConnection: idxConnection}
}

func (r *pineconeRepository) UpsertEmbeddings(ctx context.Context, embeddings []float32, metadata map[string]any, namespace string) (string, error) {

	id := fmt.Sprintf("%016x", rand.Uint64())

	metadataMap, err := structpb.NewStruct(metadata)
	if err != nil {
		return "", err
	}

	vectors := []*pinecone.Vector{
		{
			Id:       id,
			Values:   &embeddings,
			Metadata: metadataMap,
		},
	}

	_, err = r.idxConnection.WithNamespace(namespace).UpsertVectors(ctx, vectors)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *pineconeRepository) QuerySimilarVectors(ctx context.Context, queryEmbedding []float32, topK int, namespace string) ([]string, error) {

	queryRequest := &pinecone.QueryByVectorValuesRequest{
		Vector: queryEmbedding,
		TopK:   uint32(topK),
	}

	response, err := r.idxConnection.WithNamespace(namespace).QueryByVectorValues(ctx, queryRequest)
	if err != nil {
		return nil, err
	}

	embeddingIds := make([]string, 0, len(response.Matches))
	for _, match := range response.Matches {
		embeddingIds = append(embeddingIds, match.Vector.Id)
	}

	return embeddingIds, nil
}
