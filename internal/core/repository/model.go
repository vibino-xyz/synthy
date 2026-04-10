package repository

import "time"

type RepositoryProvider string

const (
	RepositoryProviderGithub RepositoryProvider = "GITHUB"
	RepositoryProviderGitLab RepositoryProvider = "GITLAB"
	RepositoryProviderOther  RepositoryProvider = "OTHER"
)

type Repository struct {
	ID             string             `json:"id" db:"id"`
	OrganizationID string             `json:"organization_id" db:"organization_id"`
	Name           string             `json:"name" db:"name"`
	Description    *string            `json:"description" db:"description"`
	DefaultBranch  string             `json:"default_branch" db:"default_branch"`
	RepositoryUrl  string             `json:"repository_url" db:"repository_url"`
	StorageUrl     string             `json:"storage_url" db:"storage_url"`
	Provider       RepositoryProvider `json:"provider" db:"provider"`
	ExternalId     string             `json:"external_id" db:"external_id"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" db:"updated_at"`
}
