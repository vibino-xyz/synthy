package analysis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	"github.com/vibino-xyz/synthy/internal/core/repository"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
)

type AnalysisPipeline struct {
	fileService        *cfile.FileService
	symbolService      *csymbol.SymbolService
	callService        *calls.CallEdgeService
	importService      *imports.ImportEdgeService
	chunkService       *chunker.ChunkService
	repoRepo           repository.RepositoryRepository
	fileRepo           cfile.FileRepository
	symbolRepo         csymbol.SymbolRepository
	chunkRepo          chunker.ChunkRepository
	vectors            embedding.PineconeRepository
	embeddingPublisher embedding.EmbeddingPublisher
}

func NewAnalysisPipeline(
	fileService *cfile.FileService,
	symbolService *csymbol.SymbolService,
	callService *calls.CallEdgeService,
	importService *imports.ImportEdgeService,
	chunkService *chunker.ChunkService,
	repoRepo repository.RepositoryRepository,
	fileRepo cfile.FileRepository,
	symbolRepo csymbol.SymbolRepository,
	chunkRepo chunker.ChunkRepository,
	vectors embedding.PineconeRepository,
	embeddingPublisher embedding.EmbeddingPublisher,
) *AnalysisPipeline {
	return &AnalysisPipeline{
		fileService:        fileService,
		symbolService:      symbolService,
		callService:        callService,
		importService:      importService,
		chunkService:       chunkService,
		repoRepo:           repoRepo,
		fileRepo:           fileRepo,
		symbolRepo:         symbolRepo,
		chunkRepo:          chunkRepo,
		vectors:            vectors,
		embeddingPublisher: embeddingPublisher,
	}
}

// RepositoryInput describes the repository being indexed. repoPath (a separate
// arg) is where its code currently lives on disk; these fields are its stable
// identity, persisted on the repository record.
type RepositoryInput struct {
	OrganizationID string
	ExternalID     string // stable id (e.g. the GitHub repo id) used for dedup
	Name           string // e.g. "vibino-xyz/commons"
	DefaultBranch  string
	RepositoryURL  string // remote url
	Provider       repository.RepositoryProvider

	// Incremental asks for only ChangedPaths/RemovedPaths to be reprocessed
	// instead of the whole repository. It is honoured only when the repository
	// has been indexed before; the first index is always full.
	Incremental  bool
	ChangedPaths []string // repo-relative, added or modified
	RemovedPaths []string // repo-relative, deleted
}

// ProcessRepository indexes repoPath into the graph.
//
// A full index replaces everything the repository owns. An incremental index
// touches only the files a push changed, leaving every other file's chunks —
// and the embeddings that were paid for to produce them — in place.
func (p *AnalysisPipeline) ProcessRepository(ctx context.Context, repoPath string, in RepositoryInput) error {
	existing, err := p.repoRepo.GetRepositoryByExternalID(ctx, in.ExternalID)
	if err != nil {
		return fmt.Errorf("check existing repository: %w", err)
	}

	repo, err := p.upsertRepositoryRecord(ctx, repoPath, in, existing)
	if err != nil {
		return err
	}

	// Incremental needs a baseline to be incremental against.
	incremental := in.Incremental && existing != nil
	if in.Incremental && existing == nil {
		slog.InfoContext(ctx, "incremental index requested for an unindexed repository, running a full index",
			"repo", in.Name)
	}

	if incremental {
		return p.processIncremental(ctx, repoPath, repo, in)
	}
	return p.processFull(ctx, repoPath, repo, existing != nil)
}

// upsertRepositoryRecord returns the repository row to index into, reusing the
// existing id. Regenerating it on every index would orphan every Pinecone vector,
// whose metadata records the repository and chunk ids at upsert time.
func (p *AnalysisPipeline) upsertRepositoryRecord(
	ctx context.Context, repoPath string, in RepositoryInput, existing *repository.Repository,
) (*repository.Repository, error) {
	defaultBranch := in.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	provider := in.Provider
	if provider == "" {
		provider = repository.RepositoryProviderOther
	}

	if existing != nil {
		existing.Name = in.Name
		existing.DefaultBranch = defaultBranch
		existing.RepositoryUrl = in.RepositoryURL
		existing.StorageUrl = repoPath
		existing.Provider = provider
		existing.UpdatedAt = time.Now().UTC()

		if err := p.repoRepo.UpdateRepository(ctx, existing); err != nil {
			return nil, fmt.Errorf("update repository: %w", err)
		}
		return existing, nil
	}

	// TODO: Move generateId to a util package
	id, err := generateID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	repo := &repository.Repository{
		ID:             id,
		OrganizationID: in.OrganizationID,
		Name:           in.Name,
		DefaultBranch:  defaultBranch,
		RepositoryUrl:  in.RepositoryURL,
		StorageUrl:     repoPath,
		Provider:       provider,
		ExternalId:     in.ExternalID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if _, err := p.repoRepo.InsertRepository(ctx, repo); err != nil {
		return nil, fmt.Errorf("insert repository: %w", err)
	}
	slog.InfoContext(ctx, "created repository record", "id", repo.ID, "name", repo.Name)
	return repo, nil
}

// processFull rebuilds the entire graph for repo.
func (p *AnalysisPipeline) processFull(ctx context.Context, repoPath string, repo *repository.Repository, reindex bool) error {
	if reindex {
		// Drop the vectors before the rows that point at them: once the files
		// are gone their embedding ids are unrecoverable and the vectors would
		// sit in Pinecone forever, matching queries for code that no longer exists.
		if err := p.deleteRepositoryVectors(ctx, repo.ID); err != nil {
			return err
		}
		slog.InfoContext(ctx, "clearing previous index", "repository_id", repo.ID)
		if err := p.fileRepo.DeleteFilesByRepositoryID(ctx, repo.ID); err != nil {
			return fmt.Errorf("delete existing files: %w", err)
		}
	}

	pkgs, err := loadPackages(ctx, repoPath)
	if err != nil {
		return err
	}

	// fileMap[absPath]*File — every file, since this is a full rebuild.
	fileMap, err := p.fileService.ProcessFiles(ctx, repo.ID, repoPath, pkgs, nil)
	if err != nil {
		return fmt.Errorf("process files: %w", err)
	}
	slog.InfoContext(ctx, "processed files", "count", len(fileMap))

	symbols, err := p.symbolService.ExtractAndSave(ctx, repo.ID, fileMap, pkgs)
	if err != nil {
		return fmt.Errorf("extract symbols: %w", err)
	}
	slog.InfoContext(ctx, "extracted symbols", "count", len(symbols))

	if err := p.buildEdges(ctx, repo.ID, fileMap, csymbol.ToFqNameMap(symbols), pkgs); err != nil {
		return err
	}

	return p.chunkAndPublish(ctx, repo.ID, fileMap, symbols)
}

// processIncremental reprocesses only the files a push touched.
//
// Deleting a file row cascades to its symbols, chunks and edges, so discarding
// a changed file's old state is a single delete — but the vectors live in
// Pinecone and have to be removed explicitly first.
//
// The dependency graph is then rebuilt across the whole repository. Edges are
// derived from the package graph and cost nothing but CPU, and a change in one
// file routinely adds or removes edges in files that did not themselves change.
// Chunks and embeddings, the expensive part, are left alone.
func (p *AnalysisPipeline) processIncremental(
	ctx context.Context, repoPath string, repo *repository.Repository, in RepositoryInput,
) error {
	existingFiles, err := p.fileRepo.GetFilesByRepositoryID(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("list existing files: %w", err)
	}

	// Repositories indexed before paths became repo-relative hold absolute paths
	// into a temp clone that no longer exists. Those can never match an incoming
	// webhook path, so every file would look new and the stale rows would linger
	// forever. Rebuild once instead; subsequent pushes are incremental again.
	for _, f := range existingFiles {
		if filepath.IsAbs(f.Path) {
			slog.InfoContext(ctx, "existing index predates repo-relative paths, running a full index",
				"repo", repo.Name)
			return p.processFull(ctx, repoPath, repo, true)
		}
	}

	byRelPath := make(map[string]*cfile.File, len(existingFiles))
	for _, f := range existingFiles {
		byRelPath[f.Path] = f
	}

	changed := goFiles(in.ChangedPaths)
	removed := goFiles(in.RemovedPaths)
	if len(changed) == 0 && len(removed) == 0 {
		slog.InfoContext(ctx, "incremental index has no Go file changes, nothing to do",
			"repo", repo.Name)
		return nil
	}

	// Discard what the push invalidated: both changed and removed files, since a
	// changed file is about to be rebuilt from its new contents.
	for _, rel := range append(append([]string{}, changed...), removed...) {
		f, ok := byRelPath[rel]
		if !ok {
			continue // new file, nothing to discard
		}
		if err := p.deleteFileVectors(ctx, f.ID); err != nil {
			return err
		}
		if err := p.fileRepo.DeleteFileByID(ctx, f.ID); err != nil {
			return fmt.Errorf("delete file %s: %w", rel, err)
		}
		delete(byRelPath, rel)
	}
	slog.InfoContext(ctx, "incremental index", "repo", repo.Name,
		"changed", len(changed), "removed", len(removed))

	pkgs, err := loadPackages(ctx, repoPath)
	if err != nil {
		return err
	}

	// Only the changed files get new rows; absolute paths, because that is how
	// the package loader identifies them.
	only := make(map[string]bool, len(changed))
	for _, rel := range changed {
		only[filepath.Join(repoPath, rel)] = true
	}
	newFiles, err := p.fileService.ProcessFiles(ctx, repo.ID, repoPath, pkgs, only)
	if err != nil {
		return fmt.Errorf("process changed files: %w", err)
	}
	slog.InfoContext(ctx, "processed changed files", "count", len(newFiles))

	newSymbols, err := p.symbolService.ExtractAndSave(ctx, repo.ID, newFiles, pkgs)
	if err != nil {
		return fmt.Errorf("extract symbols: %w", err)
	}
	slog.InfoContext(ctx, "extracted symbols", "count", len(newSymbols))

	// Edges span the whole repository, so they need every file and every symbol
	// — the surviving ones read back from the database, plus the ones just written.
	allFiles := make(map[string]*cfile.File, len(byRelPath)+len(newFiles))
	for rel, f := range byRelPath {
		allFiles[filepath.Join(repoPath, rel)] = f
	}
	for abs, f := range newFiles {
		allFiles[abs] = f
	}

	allSymbols, err := p.symbolRepo.GetSymbolsByRepositoryID(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("list symbols: %w", err)
	}

	if err := p.buildEdges(ctx, repo.ID, allFiles, csymbol.ToFqNameMap(allSymbols), pkgs); err != nil {
		return err
	}

	return p.chunkAndPublish(ctx, repo.ID, newFiles, newSymbols)
}

// buildEdges rebuilds the call and import graph for the repository from scratch.
func (p *AnalysisPipeline) buildEdges(
	ctx context.Context,
	repositoryID string,
	fileMap map[string]*cfile.File,
	symbolMap map[string]*csymbol.Symbol,
	pkgs []*packages.Package,
) error {
	callEdges, err := p.callService.BuildAndSave(ctx, repositoryID, fileMap, symbolMap, pkgs)
	if err != nil {
		return fmt.Errorf("build call edges: %w", err)
	}
	slog.InfoContext(ctx, "built call edges", "count", len(callEdges))

	importEdges, err := p.importService.BuildAndSave(ctx, repositoryID, fileMap, pkgs)
	if err != nil {
		return fmt.Errorf("build import edges: %w", err)
	}
	slog.InfoContext(ctx, "built import edges", "count", len(importEdges))
	return nil
}

// chunkAndPublish chunks each file in fileMap and queues the chunks for
// embedding and summarisation.
func (p *AnalysisPipeline) chunkAndPublish(
	ctx context.Context, repositoryID string, fileMap map[string]*cfile.File, symbols []*csymbol.Symbol,
) error {
	totalChunks := 0
	for filePath, f := range fileMap {
		fileSymbols := symbolsForFile(symbols, f.ID)
		chunks, err := p.chunkService.ProcessFile(ctx, repositoryID, f.ID, filePath, fileSymbols)
		if err != nil {
			return fmt.Errorf("process chunks for %s: %w", filePath, err)
		}
		totalChunks += len(chunks)

		// Push chunks to embedding exchange for embeddings and summary generation.
		for _, c := range chunks {
			// It's for test, so will remove this later.
			if p.embeddingPublisher == nil {
				continue
			}
			req := embedding.EmbeddingPublisherRequest{
				RepositoryId: repositoryID,
				ChunkId:      c.ID,
			}
			if err := p.embeddingPublisher.PublishEmbedding(ctx, req); err != nil {
				return fmt.Errorf("publish chunk for %s: %w", filePath, err)
			}
		}
	}
	slog.InfoContext(ctx, "processed chunks", "count", totalChunks)
	return nil
}

func loadPackages(ctx context.Context, repoPath string) ([]*packages.Package, error) {
	pkgs, err := cfile.LoadPackages(repoPath)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found at %s", repoPath)
	}
	slog.InfoContext(ctx, "loaded packages", "count", len(pkgs))
	return pkgs, nil
}

// goFiles keeps only the paths this pipeline can analyse.
func goFiles(paths []string) []string {
	var result []string
	for _, p := range paths {
		if filepath.Ext(p) == ".go" {
			result = append(result, p)
		}
	}
	return result
}

func symbolsForFile(symbols []*csymbol.Symbol, fileID string) []*csymbol.Symbol {
	var result []*csymbol.Symbol
	for _, sym := range symbols {
		if sym.FileID == fileID {
			result = append(result, sym)
		}
	}
	return result
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
