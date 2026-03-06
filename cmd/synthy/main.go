package main

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/controller/mq"
	"github.com/vibino-xyz/synthy/internal/infra/psql"
	"github.com/vibino-xyz/synthy/internal/infra/rabbimq"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			mq.NewRepositoryEventController,
		),
		fx.Provide(
			rabbimq.NewConn,
			rabbimq.NewRepositoryEventSubscriber,
		),
		fx.Provide(
			psql.NewConnection,
		),
		fx.Invoke(repositoryEventSubscriberHook),
		fx.Invoke(psql.NewConnection),
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
