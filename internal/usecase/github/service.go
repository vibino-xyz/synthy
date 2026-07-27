// Package github (usecase) orchestrates the GitHub App connection for an
// organization: building the install link, recording the installation, and
// listing the repositories/branches it can see. Authority checks live in the
// controller (from the JWT role claim); this layer assumes the caller is allowed.
package github

import (
	"context"
	"time"

	"github.com/vibino-xyz/commons/id"
	core "github.com/vibino-xyz/synthy/internal/core/github"
	"github.com/vibino-xyz/synthy/internal/core/repository"
	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
)

type Service struct {
	installations core.InstallationRepository
	client        *ghapi.Client
	publisher     repository.RepositoryEventPublisher
	repositories  repository.RepositoryRepository
}

func NewService(
	installations core.InstallationRepository,
	client *ghapi.Client,
	publisher repository.RepositoryEventPublisher,
	repositories repository.RepositoryRepository,
) *Service {
	return &Service{installations: installations, client: client, publisher: publisher, repositories: repositories}
}

// IndexedRepositoryView is one indexed repository as the dashboard renders it.
// synthy stores repository identity only (no per-repo progress/status column),
// so a row's mere existence means it has been indexed.
type IndexedRepositoryView struct {
	ID            string    `json:"id"`
	ExternalID    string    `json:"external_id"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description,omitempty"`
	DefaultBranch string    `json:"default_branch"`
	Provider      string    `json:"provider"`
	RepositoryURL string    `json:"repository_url"`
	State         string    `json:"state"`
	IndexedAt     time.Time `json:"indexed_at"`
}

// ConnectionView is the connection status the dashboard renders.
type ConnectionView struct {
	Configured     bool   `json:"configured"`
	Connected      bool   `json:"connected"`
	AccountLogin   string `json:"account_login,omitempty"`
	InstallationID int64  `json:"installation_id,omitempty"`
}

// InstallURL returns the GitHub App install link, carrying the org id as state
// so the callback can be tied back to this organization.
func (s *Service) InstallURL(orgId string) (string, error) {
	if !s.client.Configured() {
		return "", ErrNotConfigured
	}
	return s.client.InstallURL(orgId)
}

// Connect validates an installation id with GitHub and records it for the org.
func (s *Service) Connect(ctx context.Context, orgId string, installationID int64) (*ConnectionView, error) {
	if !s.client.Configured() {
		return nil, ErrNotConfigured
	}

	account, err := s.client.GetInstallationAccount(ctx, installationID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	newID, err := id.New(core.IDPrefix)
	if err != nil {
		return nil, err
	}
	installation := &core.Installation{
		Id:             newID,
		OrganizationId: orgId,
		InstallationId: installationID,
		AccountLogin:   account.Login,
		AccountType:    account.Type,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.installations.Upsert(ctx, installation); err != nil {
		return nil, err
	}

	return &ConnectionView{
		Configured:     true,
		Connected:      true,
		AccountLogin:   account.Login,
		InstallationID: installationID,
	}, nil
}

// Connection reports whether the org has the App installed.
func (s *Service) Connection(ctx context.Context, orgId string) (*ConnectionView, error) {
	view := &ConnectionView{Configured: s.client.Configured()}
	if !view.Configured {
		return view, nil
	}

	installation, err := s.installations.GetByOrganization(ctx, orgId)
	if err != nil {
		return nil, err
	}
	if installation == nil {
		return view, nil
	}

	view.Connected = true
	view.AccountLogin = installation.AccountLogin
	view.InstallationID = installation.InstallationId
	return view, nil
}

// Disconnect forgets the org's installation record. (It does not uninstall the
// App on GitHub — that is done from GitHub's settings.)
func (s *Service) Disconnect(ctx context.Context, orgId string) error {
	return s.installations.DeleteByOrganization(ctx, orgId)
}

// ListRepositories returns the repositories the org's installation can access.
func (s *Service) ListRepositories(ctx context.Context, orgId string) ([]ghapi.Repo, error) {
	installation, err := s.requireInstallation(ctx, orgId)
	if err != nil {
		return nil, err
	}
	return s.client.ListRepositories(ctx, installation.InstallationId)
}

// ListBranches returns the branches of one repo the installation can access.
func (s *Service) ListBranches(ctx context.Context, orgId, repoFullName string) ([]string, error) {
	installation, err := s.requireInstallation(ctx, orgId)
	if err != nil {
		return nil, err
	}
	return s.client.ListBranches(ctx, installation.InstallationId, repoFullName)
}

func (s *Service) IndexConnectedRepositories(ctx context.Context, orgId string) (int, error) {
	installation, err := s.requireInstallation(ctx, orgId)
	if err != nil {
		return 0, err
	}

	repos, err := s.client.ListRepositories(ctx, installation.InstallationId)
	if err != nil {
		return 0, err
	}

	published := 0
	for _, repo := range repos {
		message := &repository.IngestionMessage{
			Provider:       repository.RepositoryProviderGithub,
			EventType:      repository.IngestionEventTypeFullIndex,
			RepositoryID:   repo.ExternalID,
			RepoFullName:   repo.FullName,
			DefaultBranch:  repo.DefaultBranch,
			CloneURL:       repo.CloneURL,
			InstallationID: installation.InstallationId,
			OrganizationID: orgId,
		}
		if err := s.publisher.Publish(ctx, message); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

// ListIndexedRepositories returns the repositories synthy has indexed for the
// org, read straight from its own database (not GitHub).
func (s *Service) ListIndexedRepositories(ctx context.Context, orgId string) ([]IndexedRepositoryView, error) {
	repos, err := s.repositories.ListRepositoriesByOrganizationID(ctx, orgId)
	if err != nil {
		return nil, err
	}

	views := make([]IndexedRepositoryView, 0, len(repos))
	for _, repo := range repos {
		description := ""
		if repo.Description != nil {
			description = *repo.Description
		}
		views = append(views, IndexedRepositoryView{
			ID:            repo.ID,
			ExternalID:    repo.ExternalId,
			FullName:      repo.Name,
			Description:   description,
			DefaultBranch: repo.DefaultBranch,
			Provider:      string(repo.Provider),
			RepositoryURL: repo.RepositoryUrl,
			State:         "indexed",
			IndexedAt:     repo.UpdatedAt,
		})
	}
	return views, nil
}

func (s *Service) requireInstallation(ctx context.Context, orgId string) (*core.Installation, error) {
	if !s.client.Configured() {
		return nil, ErrNotConfigured
	}
	installation, err := s.installations.GetByOrganization(ctx, orgId)
	if err != nil {
		return nil, err
	}
	if installation == nil {
		return nil, ErrNotConnected
	}
	return installation, nil
}
