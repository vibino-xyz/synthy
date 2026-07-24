package github

import "context"

type InstallationRepository interface {
	// Upsert stores the installation for an organization, replacing any
	// previous one (an org can only have a single active installation).
	Upsert(ctx context.Context, installation *Installation) error
	GetByOrganization(ctx context.Context, organizationId string) (*Installation, error)
	DeleteByOrganization(ctx context.Context, organizationId string) error
}
