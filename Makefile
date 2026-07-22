.PHONY: run test build build-web tidy vet

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# Build the Go binary only. Does not require the frontend to be present.
build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

# Build the frontend. Separate target so `make build` never depends on web/.
build-web:
	cd web && npm ci && npm run build
