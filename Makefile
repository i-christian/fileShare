# Create a confirm target prerequisite
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# Build the application
build:
	@echo "Building..."
	@go build -ldflags="-s" -buildvcs=true -o=./main ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/main ./cmd/api

# Run the application
run:
	@echo "Running with rate limiter disabled..."
	@go run cmd/api/main.go -limiter-enabled=false

# Test the application
test:
	@echo "Testing..."
	@go test -v -cover ./... -limiter-enabled=false

# Clean the binary
clean: confirm
	@echo "Cleaning..."
	@rm -f main

# Development Docker
docker-dev-up:
	@echo "Docker building for local..."
	@docker compose -f docker-compose.yml up -d --build

# Shutdown docker
docker-down: confirm
	@echo "Destroying docker containers..."
	@docker compose down

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch
