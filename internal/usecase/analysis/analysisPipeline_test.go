package analysis

import (
	"context"
	"testing"

	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
)

func TestProcessRepository(t *testing.T) {
	repoPath := "/Users/ratnesh/Desktop/Waldo/services/argo"
	ctx := context.Background()

	db, err := psql.NewConnection()
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure a seed user and organization exist so the repository FK is satisfiable.
	orgID, err := psql.EnsureSeedData(ctx, db)
	if err != nil {
		t.Fatalf("failed to ensure seed data: %v", err)
	}

	pipeline := NewAnalysisPipeline(
		cfile.NewFileService(psql.NewFileRepository(db)),
		csymbol.NewSymbolService(psql.NewSymbolRepository(db)),
		calls.NewCallEdgeService(psql.NewCallEdgeRepository(db)),
		imports.NewImportEdgeService(psql.NewImportEdgeRepository(db)),
		chunker.NewChunkService(psql.NewChunkRepository(db)),
		psql.NewRepositoryRepository(db),
	)

	if err := pipeline.ProcessRepository(ctx, repoPath, orgID); err != nil {
		t.Fatalf("failed to process repository: %v", err)
	}

	t.Log("Successfully processed repository")
}
