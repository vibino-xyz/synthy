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
	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
)

type Service struct {
	installations core.InstallationRepository
	client        *ghapi.Client
}

func NewService(installations core.InstallationRepository, client *ghapi.Client) *Service {
	return &Service{installations: installations, client: client}
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
