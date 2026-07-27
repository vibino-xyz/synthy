package repository

import "context"

// RepositoryEventPublisher publishes repository ingestion events onto the
// message queue for the indexing worker to consume. synthy is both producer
// (on connect / reindex) and consumer of these events.
type RepositoryEventPublisher interface {
	Publish(ctx context.Context, message *IngestionMessage) error
}
