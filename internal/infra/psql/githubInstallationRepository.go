package psql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/github"
)

type githubInstallationRepository struct {
	db *pgxpool.Pool
}

func NewGithubInstallationRepository(db *pgxpool.Pool) github.InstallationRepository {
	return &githubInstallationRepository{db: db}
}

func (r *githubInstallationRepository) Upsert(ctx context.Context, i *github.Installation) error {
	const query = `
		INSERT INTO github_installation
			(id, organization_id, installation_id, account_login, account_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (organization_id) DO UPDATE SET
			installation_id = EXCLUDED.installation_id,
			account_login   = EXCLUDED.account_login,
			account_type    = EXCLUDED.account_type,
			updated_at      = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		i.Id, i.OrganizationId, i.InstallationId, i.AccountLogin, i.AccountType, i.CreatedAt,
	)
	return err
}

func (r *githubInstallationRepository) GetByOrganization(ctx context.Context, organizationId string) (*github.Installation, error) {
	const query = `
		SELECT id, organization_id, installation_id, account_login, account_type, created_at, updated_at
		FROM github_installation
		WHERE organization_id = $1`

	var i github.Installation
	err := r.db.QueryRow(ctx, query, organizationId).Scan(
		&i.Id, &i.OrganizationId, &i.InstallationId, &i.AccountLogin, &i.AccountType, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &i, nil
}

func (r *githubInstallationRepository) GetByInstallationId(ctx context.Context, installationId int64) (*github.Installation, error) {
	const query = `
		SELECT id, organization_id, installation_id, account_login, account_type, created_at, updated_at
		FROM github_installation
		WHERE installation_id = $1`

	var i github.Installation
	err := r.db.QueryRow(ctx, query, installationId).Scan(
		&i.Id, &i.OrganizationId, &i.InstallationId, &i.AccountLogin, &i.AccountType, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &i, nil
}

func (r *githubInstallationRepository) DeleteByOrganization(ctx context.Context, organizationId string) error {
	const query = `DELETE FROM github_installation WHERE organization_id = $1`
	_, err := r.db.Exec(ctx, query, organizationId)
	return err
}
