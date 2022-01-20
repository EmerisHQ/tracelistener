OBJS = $(shell find cmd -mindepth 1 -type d -execdir printf '%s\n' {} +)

TARGETS = sdk_targets.json

SHELL := /usr/bin/env bash

SETUP_VERSIONS = $(shell jq -r '.versions|map("setup-\(.)")[]'  ${TARGETS})
BUILD_VERSIONS = $(shell jq -r '.versions|map("build-\(.)")[]' ${TARGETS})
STORE_MOD_VERSIONS = $(shell jq -r '.versions|map("store-mod-\(.)")[]' ${TARGETS})
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

BASEPKG = github.com/allinbits/tracelistener
.PHONY: clean $(SETUP_VERSIONS) $(BUILD_VERSIONS)

$(BUILD_VERSIONS):
	go build -o build/tracelistener -v \
	 -tags $(shell echo $@ | sed -e 's/build-/sdk_/g' -e 's/-/_/g'),muslc \
	 -ldflags "-X main.Version=${BRANCH}-${COMMIT} -X main.SupportedSDKVersion=$(shell echo $@ | sed -e 's/build-//g' -e 's/-/_/g')" \
	 ${BASEPKG}/cmd/tracelistener

clean:
	rm -rf build
	rm go.mod go.sum | true
	cp mods/go.mod.bare ./go.mod

docker:
	docker build -t emeris/tracelistener --build-arg GIT_TOKEN=${GITHUB_TOKEN} -f Dockerfile .

$(SETUP_VERSIONS):
	cp mods/go.mod.$(shell echo $@ | sed 's/setup-//g') ./go.mod
	cp mods/go.sum.$(shell echo $@ | sed 's/setup-//g') ./go.sum

available-go-tags:
	@echo Available Go \`//go:build\' tags:
	@jq -r '.versions|map("sdk_\(.)")[]' ${TARGETS}

versions-json:
	@jq -r -c "map( { "versions": .[] } )" ${TARGETS}

$(STORE_MOD_VERSIONS):
	cp ./go.mod mods/go.mod.$(shell echo $@ | sed 's/store-mod-//g')
	cp ./go.sum mods/go.sum.$(shell echo $@ | sed 's/store-mod-//g')
