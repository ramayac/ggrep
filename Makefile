BINARY_NAME=ggrep

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
