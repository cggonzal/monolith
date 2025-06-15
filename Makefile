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

doc:
	# Documentation server running: http://localhost:6060/
	godoc -http=":6060"

# Deploy using the deploy script
deploy:
	chmod +x ./server_management/deploy.sh && ./server_management/deploy.sh

.PHONY: dev run build test deploy
