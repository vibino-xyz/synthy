package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
	"github.com/vibino-xyz/synthy/internal/infra/llm/ollama"
	"github.com/vibino-xyz/synthy/internal/infra/pinecone"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
	"github.com/vibino-xyz/synthy/internal/infra/rabbimq"
	"github.com/vibino-xyz/synthy/internal/infra/shttp"
	"github.com/vibino-xyz/synthy/internal/interface/mq"
	"github.com/vibino-xyz/synthy/internal/usecase/analysis"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Invoke(godotenv.Load),
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
			rabbimq.NewSummarySubscriber,
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

		fx.Provide(
			pinecone.NewConnection,
			pinecone.NewPineconeRepository,
		),

		// Interface controllers
		fx.Provide(
			mq.NewRepositoryEventController,
			mq.NewEmbeddingEventController,
			mq.NewSummaryEventController,
		),

		fx.Invoke(repositoryEventSubscriberHook),
		// fx.Invoke(embeddingEventControllerHook),
		fx.Invoke(summaryEventControllerHook),
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

func embeddingEventControllerHook(lc fx.Lifecycle, controller *mq.EmbeddingEventController) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := controller.Start(ctx); err != nil {
					slog.ErrorContext(ctx, "Unable to start embedding event controller", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return controller.Stop()
		},
	})
}

func summaryEventControllerHook(lc fx.Lifecycle, controller *mq.SummaryEventController) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := controller.Start(ctx); err != nil {
					slog.ErrorContext(ctx, "Unable to start summary event controller", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return controller.Stop()
		},
	})
}
