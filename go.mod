module github.com/allinbits/tracelistener

go 1.16

require (
	github.com/cockroachdb/cockroach-go/v2 v2.1.0 // indirect
	github.com/containerd/fifo v0.0.0-20210325135022-4614834762bf
	github.com/cosmos/cosmos-sdk v0.42.3
	github.com/cosmos/gaia/v4 v4.2.0 // indirect
	github.com/gin-gonic/gin v1.6.3 // indirect
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/jackc/pgx/v4 v4.11.0
	github.com/jmoiron/sqlx v1.3.2-0.20210128211550-a1d5e6473423
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/spf13/viper v1.7.1
	github.com/ugorji/go v1.2.5 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20210331175145-43e1dd70ce54 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v8 v8.18.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace google.golang.org/grpc => google.golang.org/grpc v1.33.2

replace github.com/jmoiron/sqlx => github.com/abraithwaite/sqlx v1.3.2-0.20210331022513-df9bf9884350
