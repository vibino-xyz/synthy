package repository

import (
	"fmt"

	contracts "github.com/vibino-xyz/protos/contracts/build"
)

type IngestionEventType string

const (
	IngestionEventTypeFullIndex        IngestionEventType = "FULL_INDEX"
	IngestionEventTypeIncrementalIndex IngestionEventType = "INCREMENTAL_INDEX"
)

type IngestionMessage struct {
	Provider       RepositoryProvider `json:"provider"`
	EventType      IngestionEventType `json:"event_type"`
	RepositoryID   int64              `json:"repository_id"`
	RepoFullName   string             `json:"repo_full_name"`
	DefaultBranch  string             `json:"default_branch"`
	CloneURL       string             `json:"clone_url"`
	InstallationID int64              `json:"installation_id"`
	OrganizationID string             `json:"organization_id"`

	// ChangedPaths and RemovedPaths are repo-relative and only meaningful for
	// an incremental event. They are the net effect of every commit in the push:
	// a file added and then deleted in the same push lands in RemovedPaths only.
	ChangedPaths []string `json:"changed_paths"`
	RemovedPaths []string `json:"removed_paths"`
}

func (r *IngestionMessage) FromProto(c *contracts.RepositoryEventMessage) error {
	if c == nil {
		return fmt.Errorf("ingestion message: nil proto")
	}
	if c.Repository == nil {
		return fmt.Errorf("ingestion message: missing repository")
	}

	r.Provider = convertFromProtoRepositoryProvider(c.Provider)
	r.EventType = convertFromProtoIngestionEventType(c.EventType)
	r.RepositoryID = c.Repository.Id
	r.RepoFullName = c.Repository.FullName
	r.DefaultBranch = c.Repository.DefaultBranch
	r.CloneURL = c.Repository.CloneUrl
	r.InstallationID = c.InstallationId
	r.OrganizationID = c.OrganizationId
	r.ChangedPaths, r.RemovedPaths = netFileChanges(c)
	return nil
}

func (r *IngestionMessage) ToProto() (*contracts.RepositoryEventMessage, error) {
	return &contracts.RepositoryEventMessage{
		Provider:  convertToProtoRepositoryProvider(r.Provider),
		EventType: convertToProtoIngestionEventType(r.EventType),
		Repository: &contracts.Repository{
			Id:            r.RepositoryID,
			FullName:      r.RepoFullName,
			DefaultBranch: r.DefaultBranch,
			CloneUrl:      r.CloneURL,
		},
		InstallationId: r.InstallationID,
		OrganizationId: r.OrganizationID,
	}, nil
}

// netFileChanges collapses the push's commits into one changed set and one
// removed set. Commits are applied in order and the last mention of a path wins,
// so a file modified in an early commit and deleted in a later one ends up
// removed rather than reprocessed.
func netFileChanges(c *contracts.RepositoryEventMessage) (changed, removed []string) {
	commits := c.Commits
	if len(commits) == 0 && c.HeadCommit != nil {
		commits = []*contracts.Commit{c.HeadCommit}
	}

	const (
		stateChanged = iota
		stateRemoved
	)
	state := make(map[string]int)
	order := make([]string, 0)
	mark := func(path string, s int) {
		if _, seen := state[path]; !seen {
			order = append(order, path)
		}
		state[path] = s
	}

	for _, commit := range commits {
		if commit == nil {
			continue
		}
		for _, p := range commit.Added {
			mark(p, stateChanged)
		}
		for _, p := range commit.Modified {
			mark(p, stateChanged)
		}
		for _, p := range commit.Removed {
			mark(p, stateRemoved)
		}
	}

	for _, path := range order {
		if state[path] == stateRemoved {
			removed = append(removed, path)
		} else {
			changed = append(changed, path)
		}
	}
	return changed, removed
}

func convertToProtoRepositoryProvider(provider RepositoryProvider) contracts.Provider {
	switch provider {
	case RepositoryProviderGithub:
		return contracts.Provider_GITHUB
	case RepositoryProviderGitLab:
		return contracts.Provider_GITLAB
	default:
		return contracts.Provider_UNKNOWN
	}
}

func convertToProtoIngestionEventType(eventType IngestionEventType) contracts.EventType {
	switch eventType {
	case IngestionEventTypeFullIndex:
		return contracts.EventType_FULL_INDEX
	case IngestionEventTypeIncrementalIndex:
		return contracts.EventType_INCREMENTAL_INDEX
	default:
		return contracts.EventType_UNKNOWN_EVENT
	}
}

func convertFromProtoRepositoryProvider(provider contracts.Provider) RepositoryProvider {
	switch provider {
	case contracts.Provider_GITHUB:
		return RepositoryProviderGithub
	case contracts.Provider_GITLAB:
		return RepositoryProviderGitLab
	default:
		return RepositoryProviderOther
	}
}

func convertFromProtoIngestionEventType(eventType contracts.EventType) IngestionEventType {
	switch eventType {
	case contracts.EventType_FULL_INDEX:
		return IngestionEventTypeFullIndex
	case contracts.EventType_INCREMENTAL_INDEX:
		return IngestionEventTypeIncrementalIndex
	default:
		return ""
	}
}
