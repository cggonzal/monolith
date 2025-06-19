# prevent make from printing commands when running targets
.SILENT:

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
	echo "Documentation server running: http://localhost:6060/"
	godoc -http=":6060"

# Deploy using the deploy script
deploy:
	chmod +x ./server_management/deploy.sh && ./server_management/deploy.sh


# Pass any additional arguments after "generator" through to the Go program
ARGS := $(filter-out generator,$(MAKECMDGOALS))

# Run a generator via `make generator <type> [options]`
generator:
	go run main.go generator $(ARGS)

# 'g' is an alias for 'generator'
ARGS := $(filter-out g,$(MAKECMDGOALS))
g:
	go run main.go generator $(ARGS)

# Catch-all target so extra arguments don't raise errors
%:
	@:


.PHONY: dev run build test deploy generator g doc