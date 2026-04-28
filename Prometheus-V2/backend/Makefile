.PHONY: tools fmt lint test test-integration generate sqlc oapi build verify clean

GO ?= go
SQLC ?= sqlc
OAPI ?= oapi-codegen

tools:
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	$(GO) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	$(GO) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run ./...

test:
	$(GO) test -race -cover ./...

test-integration:
	$(GO) test -race -cover -tags=integration ./...

generate: sqlc oapi

sqlc:
	$(SQLC) generate

oapi:
	$(OAPI) -config oapi-codegen.yaml api/openapi.yaml

build:
	$(GO) build -trimpath -ldflags="-s -w" -o bin/prometheus ./cmd/prometheus

verify: fmt lint test

# clean removes build artifacts and all generated code under internal/db/repo
# (sqlc emits db.go, models.go, querier.go and *.sql.go) plus the generated
# OpenAPI server stub.
clean:
	rm -rf bin/ internal/api/openapi.gen.go internal/db/repo/*.go
