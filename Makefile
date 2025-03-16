# Default target - run with air for development
.DEFAULT_GOAL := run

run:
	air

# Build the application
build:
	go build -o monolith main.go

# Run all tests
test:
	go test -v ./...

# Deploy using the deploy script
deploy:
	./server_management/deploy.sh

.PHONY: run build test deploy
