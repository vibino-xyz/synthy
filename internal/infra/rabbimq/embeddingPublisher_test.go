package rabbimq

import (
	"context"
	"testing"

	"github.com/joho/godotenv"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
)

func loadEnv() {
	if err := godotenv.Load("../../../.env"); err != nil {
		panic("Error loading .env file")
	}
}

// TestEmbeddingSummary fetches the first 10 chunks that have not yet been embedded
// and publishes each one to the embedding exchange.
func TestEmbeddingSummary(t *testing.T) {
	loadEnv()
	ctx := context.Background()

	// ── database ──────────────────────────────────────────────────────────────
	db, err := psql.NewConnection()
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	chunkRepo := psql.NewChunkRepository(db)

	chunks, err := chunkRepo.GetChunksWithoutEmbedding(ctx, 324)
	if err != nil {
		t.Fatalf("fetch chunks without embedding: %v", err)
	}
	if len(chunks) == 0 {
		t.Skip("no chunks without embedding found — run the analysis pipeline first")
	}
	t.Logf("publishing %d chunk(s) to embedding exchange", len(chunks))

	// ── message queue ──────────────────────────────────────────────────────────
	conn, err := NewConn()
	if err != nil {
		t.Fatalf("connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	publisher, err := NewEmbeddingPublisher(conn)
	if err != nil {
		t.Fatalf("create embedding publisher: %v", err)
	}

	// ── publish ────────────────────────────────────────────────────────────────
	for _, chunk := range chunks {
		req := embedding.EmbeddingPublisherRequest{
			ChunkId:      chunk.ID,
			RepositoryId: chunk.RepositoryID,
		}
		if err := publisher.PublishEmbedding(ctx, req); err != nil {
			t.Errorf("publish chunk %s: %v", chunk.ID, err)
		} else {
			t.Logf("published chunk_id=%s repository_id=%s", chunk.ID, chunk.RepositoryID)
		}
	}
}
