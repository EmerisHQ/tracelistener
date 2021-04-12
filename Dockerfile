FROM golang:1.16 as builder

ARG GIT_TOKEN

WORKDIR /app
COPY . /app

RUN go env -w GOPRIVATE=github.com/allinbits/*
RUN git config --global url."https://git:${GIT_TOKEN}@github.com".insteadOf "https://github.com"

RUN CGO_ENABLED=0 GOOS=linux GOPROXY=direct go build -v --ldflags="-s -w" -o app github.com/allinbits/tracelistener/cmd/tracelistener

FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=builder /app/app .
ENTRYPOINT ["./app"]
