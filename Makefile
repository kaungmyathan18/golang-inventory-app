.PHONY: run dev tidy test test-cover test-e2e
run:
	go run ./cmd/server

# Live reload (install: go install github.com/air-verse/air@latest)
dev:
	air

tidy:
	go mod tidy

test:
	go test ./...

test-cover:
	go test ./... -covermode=atomic -coverprofile=coverage.out
	@go tool cover -func=coverage.out | awk '/total:/ { gsub("%", "", $$3); if ($$3 + 0 < 80) { printf("coverage %.1f%% is below 80%%\n", $$3); exit 1 } printf("coverage %.1f%% meets 80%% threshold\n", $$3) }'

test-e2e:
	@set -e; \
	docker compose up -d --build app; \
	trap 'docker compose down' EXIT; \
	E2E_BASE_URL=$${E2E_BASE_URL:-http://localhost:8080} go test -tags=e2e ./test/e2e
