package repository

import contracts "github.com/vibino-xyz/protos/contracts/build"

type IngestionEventType string

const (
	IngestionEventTypeFullIndex        IngestionEventType = "FULL_INDEX"
	IngestionEventTypeIncrementalIndex IngestionEventType = "INCREMENTAL_INDEX"
)

type IngestionMessage struct {
	Provider      RepositoryProvider `json:"provider"`
	EventType     IngestionEventType `json:"event_type"`
	RepositoryID  int64              `json:"repository_id"`
	RepoFullName  string             `json:"repo_full_name"`
	DefaultBranch string             `json:"default_branch"`
	CloneURL      string             `json:"clone_url"`
}

func (r *IngestionMessage) FromProto(c *contracts.RepositoryEventMessage) error {
	r.Provider = convertFromProtoRepositoryProvider(c.Provider)
	r.EventType = convertFromProtoIngestionEventType(c.EventType)
	r.RepositoryID = c.RepositoryId
	r.RepoFullName = c.RepoFullName
	r.DefaultBranch = c.DefaultBranch
	r.CloneURL = c.CloneUrl
	return nil
}

func (r *IngestionMessage) ToProto() (*contracts.RepositoryEventMessage, error) {
	return &contracts.RepositoryEventMessage{
		Provider:      convertToProtoRepositoryProvider(r.Provider),
		EventType:     convertToProtoIngestionEventType(r.EventType),
		RepositoryId:  r.RepositoryID,
		RepoFullName:  r.RepoFullName,
		DefaultBranch: r.DefaultBranch,
		CloneUrl:      r.CloneURL,
	}, nil
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
