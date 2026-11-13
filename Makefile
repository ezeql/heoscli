.PHONY: run build fmt lint test race cover vuln tidy

# The CLI; pass its arguments with ARGS, e.g. make run ARGS="status".
run:
	go run ./cmd/heos $(ARGS)

build:
	go build -o heos ./cmd/heos

fmt:
	golangci-lint fmt

lint:
	golangci-lint run

test:
	go test ./...

race:
	go test -race -count=1 ./...

cover:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

tidy:
	go mod tidy
	git diff --exit-code go.mod
