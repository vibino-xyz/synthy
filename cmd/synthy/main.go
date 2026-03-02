package synthy

import (
	"github.com/vibino-xyz/synthy/internal/infra/psql"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			psql.NewConnection,
		),
	).Run()
}
