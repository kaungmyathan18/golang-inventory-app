.PHONY: run dev tidy test
run:
	go run ./cmd/server

# Live reload (install: go install github.com/air-verse/air@latest)
dev:
	air

tidy:
	go mod tidy

test:
	go test ./...
