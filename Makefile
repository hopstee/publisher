run:
	@go run ./cmd/publisher/main.go

build:
	@go build -o ./cmd/publisher ./cmd/publisher.main.go

test:
	@go test ./...
