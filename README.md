# ggrep

A simplified grep-like utility written in Go, designed for recursive searching within files (including traversing `.zip` archives).

## Why?

Check [HISTORY.md](HISTORY.md) for the story behind this tool.

## Features

- **Concurrent Searching:** Utilizes multiple CPU cores to scan files in parallel.
- **Transparent Zip Support:** Searches within `.zip` archives without extraction.
- **Streaming Output:** Memory-efficient scanning using a callback-based approach.
- **Standard CLI Behavior:** Writes results to `stdout`, making it pipe-friendly.

## Usage

Run `ggrep` within the directory you want to search:

```bash
./ggrep --lines=2 "ERROR" .log
```

### Flags

- `--ext`: File extension filter (e.g., `.log`, `.txt`). Can also be provided as a positional argument. Use `--all` to search everything.
- `--lines`: Number of context lines to display (default: 1).
- `--silent`: Silent mode (no console output).
- `--version`: Print version and exit.
- `--server`: Run in Server Mode (stream processing).
- `--port`: Port for server mode (default "8080").

### Server Mode

ggrep can run as a stateless stream processor, accepting data via HTTP POST.

```bash
# Start server
./ggrep --server
```

**Ingest Data:**
```bash
curl -X POST -d "some log line with ERROR in it" localhost:8080/ingest
```

**Dynamic Configuration:**
You can update the regex pattern and target URL dynamically:

```bash
# Update regex to match "CRITICAL" and forward to a webhook
curl -X POST "http://localhost:8080/config?regex=CRITICAL&target=http://example.com/webhook"
```

> **Security Note**: The regex input is limited to 100 characters.

### Usage Examples

Search for "func" in all `.go` files with 1 line of context:
```bash
./ggrep --lines=1 "func" .go
```

Search for a pattern across all files:
```bash
./ggrep "TODO" --all
```

## Makefile Targets

You can use the provided `Makefile` to manage the project:

| Target           | Description                                      |
| ---------------- | ------------------------------------------------ |
| `make build`     | Compiles the `ggrep` binary.                     |
| `make clean`     | Removes the binary and temporary files.          |
| `make test`      | Runs the Go test suite.                          |
| `make run`       | Builds and runs a local generic test.            |
| `make docker-build` | Builds the distroless Docker image.           |
| `make docker-run`| Runs the Docker container on port 8080.          |
| `make docker-compose` | Starts the service using Docker Compose.    |
| `make manual-test` | Performs a curl-based test against the running server. |

You can override the Docker repository by setting the `REPO` variable:
```bash
make docker-build REPO=myregistry.com/user
```
