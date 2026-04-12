package analysis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"

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
	embeddingPublisher embedding.EmbeddingPublisher
}

func NewAnalysisPipeline(
	fileService *cfile.FileService,
	symbolService *csymbol.SymbolService,
	callService *calls.CallEdgeService,
	importService *imports.ImportEdgeService,
	chunkService *chunker.ChunkService,
	repoRepo repository.RepositoryRepository,
	embeddingPublisher embedding.EmbeddingPublisher,
) *AnalysisPipeline {
	return &AnalysisPipeline{
		fileService:        fileService,
		symbolService:      symbolService,
		callService:        callService,
		importService:      importService,
		chunkService:       chunkService,
		repoRepo:           repoRepo,
		embeddingPublisher: embeddingPublisher,
	}
}

func (p *AnalysisPipeline) ProcessRepository(ctx context.Context, repoPath, organizationID string) error {

	// Delete existing repository
	existing, err := p.repoRepo.GetRepositoryByExternalID(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("check existing repository: %w", err)
	}
	if existing != nil {
		slog.InfoContext(ctx, "deleting existing repository index", "id", existing.ID)
		if err := p.repoRepo.DeleteRepositoryByID(ctx, existing.ID); err != nil {
			return fmt.Errorf("delete existing repository: %w", err)
		}
	}

	// TODO: Move generateId to a util package
	id, err := generateID()
	if err != nil {
		return err
	}

	name := filepath.Base(repoPath)
	repo := &repository.Repository{
		ID:             id,
		OrganizationID: organizationID,
		Name:           name,
		DefaultBranch:  "main",
		RepositoryUrl:  repoPath,
		StorageUrl:     repoPath,
		Provider:       repository.RepositoryProviderOther,
		ExternalId:     repoPath,
	}

	if _, err := p.repoRepo.InsertRepository(ctx, repo); err != nil {
		return fmt.Errorf("insert repository: %w", err)
	}
	slog.InfoContext(ctx, "created repository record", "id", repo.ID, "name", name)

	// Load packages
	pkgs, err := cfile.LoadPackages(repoPath)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages found at %s", repoPath)
	}
	slog.InfoContext(ctx, "loaded packages", "count", len(pkgs))

	// fileMap[absPath]*File.
	fileMap, err := p.fileService.ProcessFiles(ctx, repo.ID, pkgs)
	if err != nil {
		return fmt.Errorf("process files: %w", err)
	}
	slog.InfoContext(ctx, "processed files", "count", len(fileMap))

	symbols, err := p.symbolService.ExtractAndSave(ctx, repo.ID, fileMap, pkgs)
	if err != nil {
		return fmt.Errorf("extract symbols: %w", err)
	}
	// symbolMap[fqName]*Symbol.
	symbolMap := csymbol.ToFqNameMap(symbols)
	slog.InfoContext(ctx, "extracted symbols", "count", len(symbols))

	callEdges, err := p.callService.BuildAndSave(ctx, repo.ID, fileMap, symbolMap, pkgs)
	if err != nil {
		return fmt.Errorf("build call edges: %w", err)
	}
	slog.InfoContext(ctx, "built call edges", "count", len(callEdges))

	importEdges, err := p.importService.BuildAndSave(ctx, repo.ID, fileMap, pkgs)
	if err != nil {
		return fmt.Errorf("build import edges: %w", err)
	}
	slog.InfoContext(ctx, "built import edges", "count", len(importEdges))

	totalChunks := 0
	for filePath, f := range fileMap {
		fileSymbols := symbolsForFile(symbols, f.ID)
		chunks, err := p.chunkService.ProcessFile(ctx, repo.ID, f.ID, filePath, fileSymbols)
		if err != nil {
			return fmt.Errorf("process chunks for %s: %w", filePath, err)
		}
		totalChunks += len(chunks)

		// Push chunks to embedding exchange for embeddings and summary generation.
		for _, c := range chunks {
			req := embedding.EmbeddingPublisherRequest{
				RepositoryId: repo.ID,
				ChunkId:      c.ID,
			}
			// It's for test, so will remove this later.
			if p.embeddingPublisher == nil {
				continue
			}
			if err := p.embeddingPublisher.PublishEmbedding(ctx, req); err != nil {
				return fmt.Errorf("publish chunk for %s: %w", filePath, err)
			}
		}
	}
	slog.InfoContext(ctx, "processed chunks", "count", totalChunks)

	return nil
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
