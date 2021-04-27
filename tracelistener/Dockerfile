FROM golang:1.16 as builder

ARG GIT_TOKEN

RUN go env -w GOPRIVATE=github.com/allinbits/*
RUN git config --global url."https://git:${GIT_TOKEN}@github.com".insteadOf "https://github.com"

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOPROXY=direct go build -v --ldflags="-s -w" -o tracelistener github.com/allinbits/tracelistener/cmd/tracelistener

FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=builder /app/tracelistener /usr/local/bin/tracelistener
ENTRYPOINT ["/usr/local/bin/tracelistener"]
