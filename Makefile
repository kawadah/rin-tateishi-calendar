.PHONY: build test lint fmt run tidy

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	golangci-lint fmt

tidy:
	go mod tidy

# Run the full pipeline once (fetch -> archive -> ics).
run:
	go run ./cmd/calendar
