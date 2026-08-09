package file

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

// LoadPackages loads all Go packages under repoPath, retaining full type info.
func LoadPackages(repoPath string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedExportFile,
		Dir: repoPath,
	}

	return packages.Load(cfg, "./...")
}

type FileService struct {
	repo FileRepository
}

func NewFileService(repo FileRepository) *FileService {
	return &FileService{repo: repo}
}

// ProcessFiles derives a File record for every Go source file found in pkgs,
// persists each one, and returns a map from absolute file path to the saved File.
//
// only, when non-nil, restricts processing to those absolute paths — an
// incremental index reprocesses just the files a push touched.
//
// The map is keyed by absolute path because that is what go/token reports, but
// File.Path is stored relative to repoRoot: the clone lives in a fresh temp
// directory on every run, so an absolute path could never match the same file
// across two indexes.
func (s *FileService) ProcessFiles(ctx context.Context, repositoryID, repoRoot string, pkgs []*packages.Package, only map[string]bool) (map[string]*File, error) {
	fileMap := make(map[string]*File)

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			filePath := pkg.Fset.File(f.Pos()).Name()
			if _, ok := fileMap[filePath]; ok {
				continue
			}
			if only != nil && !only[filePath] {
				continue
			}

			relPath, err := RelPath(repoRoot, filePath)
			if err != nil {
				return nil, err
			}

			checksum, err := computeChecksum(filePath)
			if err != nil {
				return nil, fmt.Errorf("compute checksum for %s: %w", filePath, err)
			}

			ext := filepath.Ext(filePath)
			lang := LanguageOther
			if ext == ".go" {
				lang = LanguageGo
			}

			id, err := generateID()
			if err != nil {
				return nil, err
			}

			now := time.Now().UTC()
			file := &File{
				ID:           id,
				RepositoryID: repositoryID,
				Path:         relPath,
				Name:         filepath.Base(filePath),
				Extension:    strings.TrimPrefix(ext, "."),
				Checksum:     checksum,
				Language:     lang,
				IsBinary:     false,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			if err := s.repo.InsertFile(ctx, file); err != nil {
				return nil, fmt.Errorf("insert file %s: %w", filePath, err)
			}

			fileMap[filePath] = file
		}
	}

	return fileMap, nil
}

// RelPath converts an absolute path inside the clone to the repo-relative form
// stored on File.Path, so it matches the paths a provider webhook reports.
func RelPath(repoRoot, absPath string) (string, error) {
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("relativise %s against %s: %w", absPath, repoRoot, err)
	}
	return filepath.ToSlash(rel), nil
}

func computeChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
