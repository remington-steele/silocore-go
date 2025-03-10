.PHONY: migrate migrate-down migrate-force build-migrate build-server run-server build-css build-templ

# Build the migration tool
build-migrate:
	go build -o bin/migrate cmd/migrate/main.go

# Build the server
build-server:
	go build -o bin/server cmd/server/main.go

# Run the server
run-server: build-server
	./bin/server

# Run migrations up
migrate: build-migrate
	./bin/migrate

# Run migrations down
migrate-down: build-migrate
	./bin/migrate -down

# Run a specific number of migrations up
migrate-steps: build-migrate
	./bin/migrate -steps $(steps)

# Run a specific number of migrations down
migrate-down-steps: build-migrate
	./bin/migrate -down -steps $(steps)

# Build CSS with Tailwind
build-css:
	./bin/tailwindcss -i ./internal/static/css/input.css -o ./internal/static/css/output.css --minify

# Watch CSS changes
watch-css:
	./bin/tailwindcss -i ./internal/static/css/input.css -o ./internal/static/css/output.css --watch

# Build templ templates
build-templ:
	templ generate

# Watch templ changes
watch-templ:
	templ generate --watch

# Clean build artifacts
clean:
	rm -rf bin/

# Build all binaries
build: build-migrate build-server build-css build-templ

# Default target
all: build

# Run the application (with automatic migrations)
run: build
	./bin/server 