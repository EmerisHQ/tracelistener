FROM golang:1.17-alpine3.14 as builder

ARG GIT_TOKEN
ARG SDK_TARGET

RUN set -eux; apk add --no-cache ca-certificates build-base git jq bash findutils

RUN go env -w GOPRIVATE=github.com/emerishq,github.com/allinbits
RUN git config --global url."https://git:${GIT_TOKEN}@github.com".insteadOf "https://github.com"

WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
RUN make clean

RUN CGO_ENABLED=0 GOPROXY=direct make setup-${SDK_TARGET}
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOPROXY=direct make build-${SDK_TARGET}

FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app

COPY --from=builder /app/build/tracelistener /usr/local/bin/tracelistener
USER app
ENTRYPOINT ["/usr/local/bin/tracelistener"]
