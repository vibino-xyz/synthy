package main

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
	"github.com/vibino-xyz/synthy/internal/infra/llm/ollama"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
	"github.com/vibino-xyz/synthy/internal/infra/rabbimq"
	"github.com/vibino-xyz/synthy/internal/infra/shttp"
	"github.com/vibino-xyz/synthy/internal/interface/mq"
	"github.com/vibino-xyz/synthy/internal/usecase/analysis"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Infrastructure — HTTP client (shared)
		fx.Provide(
			shttp.NewClient,
		),

		// Infrastructure — LLM
		fx.Provide(
			ollama.NewLLMClient,
			ollama.NewEmbeddingClient,
		),

		// Infrastructure — message queue
		fx.Provide(
			rabbimq.NewConn,
			rabbimq.NewIngestionSubscriber,
			rabbimq.NewEmbeddingPublisher,
			rabbimq.NewEmbeddingSubscriber,
		),

		// Infrastructure — database
		fx.Provide(
			psql.NewConnection,
			psql.NewRepositoryRepository,
			psql.NewFileRepository,
			psql.NewSymbolRepository,
			psql.NewCallEdgeRepository,
			psql.NewImportEdgeRepository,
			psql.NewChunkRepository,
		),

		// Domain services
		fx.Provide(
			cfile.NewFileService,
			csymbol.NewSymbolService,
			calls.NewCallEdgeService,
			imports.NewImportEdgeService,
			chunker.NewChunkService,
		),

		// Use-cases
		fx.Provide(
			analysis.NewAnalysisPipeline,
		),

		// Interface controllers
		fx.Provide(
			mq.NewRepositoryEventController,
		),

		fx.Invoke(repositoryEventSubscriberHook),
		fx.Invoke(embeddingSubscriberHook),
	).Run()
}

func repositoryEventSubscriberHook(lc fx.Lifecycle, controller *mq.RepositoryEventController) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := controller.Start(ctx); err != nil {
					slog.ErrorContext(ctx, "Unable to start size tier processor worker", "error", err)
				}
			}()
			return nil
		},

		OnStop: func(ctx context.Context) error {
			cancel()
			return controller.Stop()
		},
	})
}

func embeddingSubscriberHook(lc fx.Lifecycle, sub embedding.EmbeddingSubscriber) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				messages, err := sub.Subscribe(ctx)
				if err != nil {
					slog.Error("failed to subscribe to embedding events", "error", err)
					return
				}
				for msg := range messages {
					slog.Info("received embedding event", "event", msg.Message)
					if err := msg.Ack(); err != nil {
						slog.Error("failed to ack embedding event", "error", err)
					}
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return sub.Close()
		},
	})
}
