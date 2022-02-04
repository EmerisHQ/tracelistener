OBJS = $(shell find cmd -mindepth 1 -type d -execdir printf '%s\n' {} +)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
BASEPKG = github.com/allinbits/tracelistener
EXTRAFLAGS :=

.PHONY: $(OBJS) clean generate-swagger

all: $(OBJS)

clean:
	@rm -rf build docs/swagger.* docs/docs.go

generate-swagger:
	go generate ${BASEPKG}/docs
	@rm docs/docs.go

lint:
	golangci-lint run ./...

test:
	go test -v -race ./... -cover

$(OBJS):
	go build -o build/$@ -ldflags='-X main.Version=${BRANCH}-${COMMIT}' ${EXTRAFLAGS} ${BASEPKG}/cmd/$@

generate-test-data:
	./tracelistener/scripts/multichain_setup_script.sh
	./tracelistener/scripts/generate_txs.sh
	./tracelistener/scripts/relayer_script.sh
	./tracelistener/scripts/stop_daemon.sh

