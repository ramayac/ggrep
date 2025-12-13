BINARY_NAME=ggrep

# Default target
all: build

build:
	go build -o $(BINARY_NAME) main.go

# Clean output file and binary
clean:
	rm -f $(BINARY_NAME) out.txt

# Run generic test
run: build
	./$(BINARY_NAME) "TODO" .go 1

# Cross compilation for other platforms if needed
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux main.go

build-mac:
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-mac main.go

