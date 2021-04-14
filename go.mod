module github.com/allinbits/tracelistener

go 1.16

require (
	github.com/allinbits/navigator-utils v0.0.0-20210414132536-93d878231418
	github.com/containerd/fifo v0.0.0-20210325135022-4614834762bf
	github.com/cosmos/cosmos-sdk v0.42.3
	github.com/cosmos/gaia/v4 v4.2.0
	github.com/go-playground/validator/v10 v10.5.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/tendermint/liquidity v1.2.3
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20210331175145-43e1dd70ce54 // indirect
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace google.golang.org/grpc => google.golang.org/grpc v1.33.2

replace github.com/jmoiron/sqlx => github.com/abraithwaite/sqlx v1.3.2-0.20210331022513-df9bf9884350

replace github.com/allinbits/navigator-utils => /Users/gsora/Documents/Tendermint/navigator/navigator-utils
