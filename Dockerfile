# --- BUILD STAGE ---
FROM golang:1.24-alpine AS builder

# Install certs for HTTPS forwarding
RUN apk add --no-cache ca-certificates

WORKDIR /build
COPY . .

# Build flags: 
# CGO_ENABLED=0 (static binary), -s -w (shrink size by removing debug info)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ggrep .

# --- FINAL STAGE ---
FROM scratch

# Import certs from builder so we can talk to HTTPS endpoints
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy the binary
COPY --from=builder /build/ggrep /ggrep

# Expose ports
EXPOSE 8080 

# Default buffer size
ENV GGREP_BUFFER_SIZE=1000 

# Default to server mode
ENTRYPOINT ["/ggrep", "--server"]
