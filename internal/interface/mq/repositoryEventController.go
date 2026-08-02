package mq

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/vibino-xyz/synthy/internal/core/github"
	"github.com/vibino-xyz/synthy/internal/core/repository"
	gitx "github.com/vibino-xyz/synthy/internal/infra/git"
	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
	"github.com/vibino-xyz/synthy/internal/usecase/analysis"
)

// RepositoryEventController consumes repository ingestion events and runs the
// full-index pipeline: mint an installation token, clone the repo, then hand
// the working tree to the analysis pipeline.
type RepositoryEventController struct {
	ingestionEventSubscriber     repository.IngestionSubscriber
	pipeline                     *analysis.AnalysisPipeline
	github                       *ghapi.Client
	cloner                       *gitx.Cloner
	githubInstallationRepository github.InstallationRepository
}

func NewRepositoryEventController(
	ingestionEventSubscriber repository.IngestionSubscriber,
	pipeline *analysis.AnalysisPipeline,
	github *ghapi.Client,
	cloner *gitx.Cloner,
	githubInstallationRepository github.InstallationRepository,
) *RepositoryEventController {
	return &RepositoryEventController{
		ingestionEventSubscriber:     ingestionEventSubscriber,
		pipeline:                     pipeline,
		github:                       github,
		cloner:                       cloner,
		githubInstallationRepository: githubInstallationRepository,
	}
}

func (c *RepositoryEventController) Start(ctx context.Context) error {
	messages, err := c.ingestionEventSubscriber.Subscribe(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to repository events", "error", err)
		return err
	}

	for msg := range messages {
		if err := c.handle(ctx, msg.Message); err != nil {
			slog.ErrorContext(ctx, "failed to index repository", "repo", msg.Message.RepoFullName, "error", err)
			// Don't requeue: a bad clone/parse will keep failing. The queue's
			// dead-letter exchange captures it for inspection.
			if nackErr := msg.Nack(false); nackErr != nil {
				slog.ErrorContext(ctx, "failed to nack repository event", "error", nackErr)
			}
			continue
		}
		if ackErr := msg.Ack(); ackErr != nil {
			slog.ErrorContext(ctx, "failed to ack repository event", "error", ackErr)
		}
	}
	return nil
}

func (c *RepositoryEventController) handle(ctx context.Context, m *repository.IngestionMessage) error {
	slog.InfoContext(ctx, "indexing repository", "repo", m.RepoFullName, "branch", m.DefaultBranch)

	// Private repos need an installation token to clone. Tokenless jobs clone
	// anonymously (public repos / a future webhook without an installation).
	token := ""
	if m.InstallationID != 0 {
		minted, err := c.github.InstallationToken(ctx, m.InstallationID)
		if err != nil {
			return fmt.Errorf("mint installation token: %w", err)
		}
		token = minted
	}

	if m.OrganizationID == "" && m.InstallationID != 0 {
		installation, err := c.githubInstallationRepository.GetByInstallationId(ctx, m.InstallationID)
		if err != nil {
			return fmt.Errorf("get installation by id: %w", err)
		}
		m.OrganizationID = installation.OrganizationId
	}

	path, cleanup, err := c.cloner.Clone(ctx, m.CloneURL, m.DefaultBranch, m.RepoFullName, token)
	if err != nil {
		return fmt.Errorf("clone repository: %w", err)
	}
	defer cleanup()

	if err := c.pipeline.ProcessRepository(ctx, path, analysis.RepositoryInput{
		OrganizationID: m.OrganizationID,
		ExternalID:     strconv.FormatInt(m.RepositoryID, 10),
		Name:           m.RepoFullName,
		DefaultBranch:  m.DefaultBranch,
		RepositoryURL:  m.CloneURL,
		Provider:       m.Provider,
	}); err != nil {
		return fmt.Errorf("process repository: %w", err)
	}

	slog.InfoContext(ctx, "indexed repository", "repo", m.RepoFullName)
	return nil
}

func (c *RepositoryEventController) Stop() error {
	return c.ingestionEventSubscriber.Close()
}
