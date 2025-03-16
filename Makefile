# Default target - run with air for development
.DEFAULT_GOAL := dev

# Run the application with air for development to allow for hot reloading
dev:
	air

# Run the application
run:
	go run main.go

# Build the application
build:
	go build -o monolith main.go

# Run all tests
test:
	go test -v ./...

# Deploy using the deploy script
deploy:
	./server_management/deploy.sh

.PHONY: dev run build test deploy
