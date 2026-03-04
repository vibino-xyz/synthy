package events

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	repositoryv1 "github.com/vibino-xyz/protos/contracts/build/go/repository/v1"
)

type RepositoryEventSubscriber interface {
	Subscribe(ctx context.Context) (<-chan *ProcessRepositoryEventMessage, error)
	Close() error
}

type ProcessRepositoryEventMessage struct {
	Message  RepositoryEventMessage
	Delivery amqp.Delivery
}

func (m *ProcessRepositoryEventMessage) Ack() error {
	return m.Delivery.Ack(false)
}

func (m *ProcessRepositoryEventMessage) Nack(requeue bool) error {
	return m.Delivery.Nack(false, requeue)
}

// TODO: Move these types to their separate file
type RepositoryProvider string

const (
	RepositoryProviderGithub RepositoryProvider = "GITHUB"
	RepositoryProviderGitlab RepositoryProvider = "GITLAB"
)

type RepositoryEventType string

const (
	RepositoryEventTypeFullIndex        RepositoryEventType = "FULL_INDEX"
	RepositoryEventTypeIncrementalIndex RepositoryEventType = "INCREMENTAL_INDEX"
)

type RepositoryEventMessage struct {
	Provider      RepositoryProvider  `json:"provider"`
	EventType     RepositoryEventType `json:"event_type"`
	RepositoryID  int64               `json:"repository_id"`
	RepoFullName  string              `json:"repo_full_name"`
	DefaultBranch string              `json:"default_branch"`
	CloneURL      string              `json:"clone_url"`
}

func FromProtos(message *repositoryv1.RepositoryEventMessage) RepositoryEventMessage {
	return RepositoryEventMessage{
		Provider:      RepositoryProvider(message.Provider),
		EventType:     RepositoryEventType(message.EventType),
		RepositoryID:  message.RepositoryId,
		RepoFullName:  message.RepoFullName,
		DefaultBranch: message.DefaultBranch,
		CloneURL:      message.CloneUrl,
	}
}
