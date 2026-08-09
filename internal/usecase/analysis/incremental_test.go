package analysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
)

// writeFixtureRepo lays down a tiny two-file Go module to index.
func writeFixtureRepo(t *testing.T, keepBody string) string {
	t.Helper()
	dir := t.TempDir()

	write := func(name, content string) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("go.mod", "module fixture\n\ngo 1.21\n")
	// No trailing newline: this is the shape that used to produce blank chunks.
	write("keep/keep.go", "package keep\n\nfunc Keep() string {\n\treturn \""+keepBody+"\"\n}")
	write("edit/edit.go", "package edit\n\nfunc Edit() string {\n\treturn \"before\"\n}")

	return dir
}

// TestIncrementalIndexPreservesUnchangedFiles indexes a repository, then runs an
// incremental index for a single changed file, and asserts that the untouched
// file keeps its original chunk rows (and therefore its embeddings) while the
// changed file's chunks are rebuilt from the new source.
func TestIncrementalIndexPreservesUnchangedFiles(t *testing.T) {
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Skipf("no .env: %v", err)
	}
	ctx := context.Background()

	db, err := psql.NewConnection()
	if err != nil {
		t.Skipf("no database: %v", err)
	}
	// Registered first so it runs last: cleanups are LIFO and the row cleanup
	// below still needs an open pool.
	t.Cleanup(db.Close)

	orgID, err := psql.EnsureSeedData(ctx, db)
	if err != nil {
		t.Fatalf("seed data: %v", err)
	}

	fileRepo := psql.NewFileRepository(db)
	symbolRepo := psql.NewSymbolRepository(db)
	chunkRepo := psql.NewChunkRepository(db)
	repoRepo := psql.NewRepositoryRepository(db)

	pipeline := NewAnalysisPipeline(
		cfile.NewFileService(fileRepo),
		csymbol.NewSymbolService(symbolRepo),
		calls.NewCallEdgeService(psql.NewCallEdgeRepository(db)),
		imports.NewImportEdgeService(psql.NewImportEdgeRepository(db)),
		chunker.NewChunkService(chunkRepo),
		repoRepo,
		fileRepo,
		symbolRepo,
		chunkRepo,
		nil, // vectors: nothing was embedded, so nothing to delete
		nil, // publisher: not exercised here
	)

	const externalID = "test-incremental-fixture"
	in := RepositoryInput{
		OrganizationID: orgID,
		ExternalID:     externalID,
		Name:           "fixture/incremental",
		DefaultBranch:  "main",
		RepositoryURL:  "https://example.invalid/fixture.git",
	}

	// Full index.
	repoPath := writeFixtureRepo(t, "original")
	if err := pipeline.ProcessRepository(ctx, repoPath, in); err != nil {
		t.Fatalf("full index: %v", err)
	}

	repo, err := repoRepo.GetRepositoryByExternalID(ctx, externalID)
	if err != nil || repo == nil {
		t.Fatalf("repository not found after full index: %v", err)
	}
	t.Cleanup(func() {
		if err := repoRepo.DeleteRepositoryByID(ctx, repo.ID); err != nil {
			t.Errorf("cleanup: delete fixture repository: %v", err)
		}
	})
	repoIDAfterFull := repo.ID

	chunksBefore, err := chunkRepo.GetChunksByRepositoryID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("list chunks: %v", err)
	}
	if len(chunksBefore) == 0 {
		t.Fatal("full index produced no chunks")
	}
	for _, c := range chunksBefore {
		if c.Content == "" {
			t.Errorf("chunk %s has empty content", c.ID)
		}
	}

	keepChunkIDs := map[string]bool{}
	var editChunkIDs []string
	filesBefore, err := fileRepo.GetFilesByRepositoryID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	for _, f := range filesBefore {
		if filepath.IsAbs(f.Path) {
			t.Errorf("file path %q is absolute, expected repo-relative", f.Path)
		}
		fileChunks, err := chunkRepo.GetChunksByFileID(ctx, f.ID)
		if err != nil {
			t.Fatalf("chunks for %s: %v", f.Path, err)
		}
		for _, c := range fileChunks {
			switch f.Path {
			case "keep/keep.go":
				keepChunkIDs[c.ID] = true
			case "edit/edit.go":
				editChunkIDs = append(editChunkIDs, c.ID)
			}
		}
	}
	if len(keepChunkIDs) == 0 || len(editChunkIDs) == 0 {
		t.Fatalf("fixture did not chunk as expected: keep=%d edit=%d", len(keepChunkIDs), len(editChunkIDs))
	}

	// Incremental index: only edit/edit.go changed.
	repoPath2 := writeFixtureRepo(t, "original")
	if err := os.WriteFile(
		filepath.Join(repoPath2, "edit/edit.go"),
		[]byte("package edit\n\nfunc Edit() string {\n\treturn \"after\"\n}"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	incremental := in
	incremental.Incremental = true
	incremental.ChangedPaths = []string{"edit/edit.go"}
	if err := pipeline.ProcessRepository(ctx, repoPath2, incremental); err != nil {
		t.Fatalf("incremental index: %v", err)
	}

	repoAfter, err := repoRepo.GetRepositoryByExternalID(ctx, externalID)
	if err != nil || repoAfter == nil {
		t.Fatalf("repository missing after incremental: %v", err)
	}
	if repoAfter.ID != repoIDAfterFull {
		t.Errorf("repository id changed across index: %s -> %s", repoIDAfterFull, repoAfter.ID)
	}

	chunksAfter, err := chunkRepo.GetChunksByRepositoryID(ctx, repoAfter.ID)
	if err != nil {
		t.Fatalf("list chunks after: %v", err)
	}

	survived, rebuilt := 0, 0
	sawAfterBody := false
	for _, c := range chunksAfter {
		if keepChunkIDs[c.ID] {
			survived++
		}
		for _, old := range editChunkIDs {
			if c.ID == old {
				t.Errorf("chunk %s from the changed file survived the incremental index", c.ID)
			}
		}
		if c.Content == "" {
			t.Errorf("chunk %s has empty content after incremental", c.ID)
		}
		if contains(c.Content, `"after"`) {
			sawAfterBody = true
			rebuilt++
		}
		if contains(c.Content, `"before"`) {
			t.Errorf("chunk %s still holds pre-change source", c.ID)
		}
	}

	if survived != len(keepChunkIDs) {
		t.Errorf("unchanged file lost chunks: %d of %d survived", survived, len(keepChunkIDs))
	}
	if !sawAfterBody {
		t.Errorf("changed file was not reindexed with its new content (rebuilt=%d)", rebuilt)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
