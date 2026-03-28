module github.com/vibino-xyz/synthy

go 1.24.0

replace github.com/vibino-xyz/protos => ../protos

require (
	github.com/jackc/pgx/v5 v5.8.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/vibino-xyz/protos v0.0.0-00010101000000-000000000000
	go.uber.org/fx v1.24.0
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/mod v0.33.0 // indirect

require (
	github.com/golang-migrate/migrate/v4 v4.19.1 // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/tools v0.42.0
)
