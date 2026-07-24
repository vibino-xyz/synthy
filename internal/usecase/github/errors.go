package github

import (
	"errors"

	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
)

var (
	// ErrNotConfigured re-exports the infra sentinel so the API layer can map
	// it without importing the infra package.
	ErrNotConfigured = ghapi.ErrNotConfigured
	// ErrNotConnected means the org has not installed the GitHub App yet.
	ErrNotConnected = errors.New("github is not connected for this organization")
)
