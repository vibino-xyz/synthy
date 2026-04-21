package analysis

import (
	"context"
	"testing"

	"github.com/joho/godotenv"
	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
	"github.com/vibino-xyz/synthy/internal/infra/rabbimq"
)

func loadEnv() {
	if err := godotenv.Load("../../../.env"); err != nil {
		panic("Error loading .env file")
	}
}

func TestProcessRepository(t *testing.T) {
	loadEnv()
	repoPath := "/Users/ratnesh/Desktop/Waldo/services/argo"
	ctx := context.Background()

	db, err := psql.NewConnection()
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	mqConn, err := rabbimq.NewConn()
	if err != nil {
		t.Fatalf("failed to connect to message queue: %v", err)
	}
	defer mqConn.Close()

	// Ensure a seed user and organization exist so the repository FK is satisfiable.
	orgID, err := psql.EnsureSeedData(ctx, db)
	if err != nil {
		t.Fatalf("failed to ensure seed data: %v", err)
	}

	embeddingPublisher, err := rabbimq.NewEmbeddingPublisher(mqConn)
	if err != nil {
		t.Fatalf("failed to create embedding publisher: %v", err)
	}

	pipeline := NewAnalysisPipeline(
		cfile.NewFileService(psql.NewFileRepository(db)),
		csymbol.NewSymbolService(psql.NewSymbolRepository(db)),
		calls.NewCallEdgeService(psql.NewCallEdgeRepository(db)),
		imports.NewImportEdgeService(psql.NewImportEdgeRepository(db)),
		chunker.NewChunkService(psql.NewChunkRepository(db)),
		psql.NewRepositoryRepository(db),
		embeddingPublisher, // Use a no-op publisher for testing
	)

	if err := pipeline.ProcessRepository(ctx, repoPath, orgID); err != nil {
		t.Fatalf("failed to process repository: %v", err)
	}

	t.Log("Successfully processed repository")
}
