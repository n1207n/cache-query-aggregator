FROM golang:1.24.3-alpine3.22 AS base
WORKDIR /app
RUN apk add --no-cache git build-base

FROM base AS deps
COPY go.mod go.sum ./
RUN go mod download

FROM deps AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /app/main cmd/server/main.go

FROM base AS tools
RUN go install github.com/air-verse/air@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

RUN cp /go/bin/air /usr/local/bin/air && \
    cp /go/bin/dlv /usr/local/bin/dlv && \
    cp /go/bin/migrate /usr/local/bin/migrate && \
    cp /go/bin/sqlc /usr/local/bin/sqlc

FROM tools AS dev-common
COPY . .
COPY --from=deps /app/go.mod /app/go.mod
COPY --from=deps /app/go.sum /app/go.sum

FROM alpine:latest AS prod
WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client

COPY --from=builder /app/main /app/main
COPY --from=tools /usr/local/bin/migrate /usr/local/bin/migrate
COPY --from=tools /usr/local/bin/sqlc /usr/local/bin/sqlc

COPY ./sqlc.yaml ./sqlc.yaml
COPY ./db/migration ./db/migration
COPY ./db/queries ./db/queries

COPY ./scripts/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/main"]
