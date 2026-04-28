.PHONY: tools fmt fmt-check lint test test-integration generate sqlc oapi build verify clean

GO ?= go
SQLC ?= sqlc
OAPI ?= oapi-codegen

tools:
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	$(GO) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	$(GO) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# fmt mutates files to apply gofmt formatting. Use this locally when you want
# to fix formatting; CI should call fmt-check instead so unformatted code fails
# the build instead of being silently rewritten.
fmt:
	$(GO) fmt ./...

# fmt-check exits non-zero if any file needs reformatting. Suitable for CI.
fmt-check:
	@unformatted="$$(gofmt -l . 2>/dev/null)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Files need gofmt:"; echo "$$unformatted"; exit 1; \
	fi

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

# verify is CI-safe: fmt-check fails on unformatted files instead of rewriting
# them, so unformatted code can't sneak into the build.
verify: fmt-check lint test

# clean removes build artifacts and the explicit list of generated files so
# hand-written code in internal/db/repo (e.g. integration test helpers) is not
# accidentally deleted by a glob.
clean:
	rm -rf bin/
	rm -f internal/api/openapi.gen.go
	rm -f internal/db/repo/db.go internal/db/repo/models.go internal/db/repo/querier.go
	rm -f internal/db/repo/*.sql.go
