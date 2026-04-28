.PHONY: tools generate verify build build-frontend dev dev-up dev-down clean

tools:
	$(MAKE) -C backend tools
	cd frontend && npm install

generate:
	$(MAKE) -C backend generate
	cd frontend && npm run generate:api

verify:
	$(MAKE) -C backend verify
	cd frontend && npm run type-check && npm run lint && npm run test

# build-frontend produces the Vite bundle and copies it into
# backend/internal/web/dist so that go build can embed it.
#
# NOTE: The repo tracks a placeholder backend/internal/web/dist/index.html via a
# whitelist .gitignore rule. Running this target overwrites that placeholder
# with the real Vite-generated index.html, which will show up as an unstaged
# change in `git status`. Don't commit it -- `make clean` restores the tree,
# and the Docker build (Task 11) is the canonical artifact path.
build-frontend:
	cd frontend && npm run build
	rm -rf backend/internal/web/dist
	cp -r frontend/dist backend/internal/web/dist

build: build-frontend
	$(MAKE) -C backend build

dev-up:
	docker compose up -d postgres redis

dev-down:
	docker compose down

dev:
	@echo "Run two terminals:"
	@echo "  1) cd backend && go run ./cmd/prometheus"
	@echo "  2) cd frontend && npm run dev"

clean:
	$(MAKE) -C backend clean
	rm -rf frontend/dist frontend/node_modules backend/internal/web/dist
