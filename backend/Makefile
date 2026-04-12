.PHONY: run build tidy test lint

## run: start the server locally (requires DB + Redis — run `make infra` first from repo root)
run:
	go run ./cmd/server

## build: compile to ./server binary
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o server ./cmd/server

## tidy: download and tidy Go modules (also generates go.sum)
tidy:
	go mod tidy

## test: run all tests
test:
	go test ./...

## lint: run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...
