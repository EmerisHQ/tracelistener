module github.com/allinbits/tracelistener

go 1.17

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require (
	github.com/allinbits/demeris-backend-models v0.0.0-20211018093214-0546d958f4d9
	github.com/allinbits/emeris-utils v1.0.0
)
