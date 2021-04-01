module github.com/allinbits/tracelistener

go 1.16

require (
	github.com/containerd/fifo v0.0.0-20210325135022-4614834762bf
	github.com/cosmos/cosmos-sdk v0.42.3
	github.com/go-playground/validator/v10 v10.4.1
	github.com/jackc/pgx/v4 v4.11.0
	github.com/jmoiron/sqlx v1.3.1
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/spf13/viper v1.7.1
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace google.golang.org/grpc => google.golang.org/grpc v1.33.2
