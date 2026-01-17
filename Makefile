BINARY_NAME=ggrep
REPO ?= docker.io/library

# Default target
all: build

build:
	go build -o $(BINARY_NAME) .

# Clean output file and binary
clean:
	rm -f $(BINARY_NAME) out.txt

# Run generic test
run: build
	./$(BINARY_NAME) "TODO" .go 1

# Cross compilation for other platforms if needed
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux .

build-mac:
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-mac .


# Run tests
test:
	go test -v ./...

# Docker targets
docker-build:
	docker build -t $(REPO)/$(BINARY_NAME):distroless .

docker-run:
	docker run -p 8080:8080 $(REPO)/$(BINARY_NAME):distroless

docker-compose:
	REPO=$(REPO) docker compose up

manual-test:
	@echo "Configuring grep service to match 'ERROR'..."
	curl -X POST "http://localhost:8080/config?regex=ERROR"
	@echo "\nSending logs..."
	curl -X POST -d "System OK" http://localhost:8080/ingest
	curl -X POST -d "System ERROR: Critical failure" http://localhost:8080/ingest
	@echo "\nCheck docker logs to see the filtered output."
