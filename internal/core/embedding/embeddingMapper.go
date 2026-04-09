package embedding

import contracts "github.com/vibino-xyz/protos/contracts/build"

type EmbeddingProcessMessage struct {
	ChunkID      string `json:"chunk_id"`
	RepositoryID string `json:"repository_id"`
}

func (p *EmbeddingProcessMessage) FromProto(c *contracts.EmbeddingEventMessage) error {
	p.ChunkID = c.ChunkId
	p.RepositoryID = c.RepositoryId
	return nil
}

func (p *EmbeddingPublisherRequest) ToProto() (*contracts.EmbeddingEventMessage, error) {
	return &contracts.EmbeddingEventMessage{
		ChunkId:      p.ChunkId,
		RepositoryId: p.RepositoryId,
	}, nil
}
