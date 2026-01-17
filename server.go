package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
)

// ServerConfig holds the dynamic parts of our "Grep" service
type ServerConfig struct {
	Regex     *regexp.Regexp
	TargetURL string
}

var globalConfig atomic.Value
var configMutex sync.Mutex

// In-memory buffer for matches
var matchBuffer []string
var bufferMutex sync.Mutex

var maxBufferSize = 1000

func startServer(port string) error {
	// Parse GGREP_BUFFER_SIZE env var
	if val := os.Getenv("GGREP_BUFFER_SIZE"); val != "" {
		if s, err := strconv.Atoi(val); err == nil && s > 0 {
			maxBufferSize = s
			log.Printf("Configured max buffer size: %d", maxBufferSize)
		} else {
			log.Printf("Invalid GGREP_BUFFER_SIZE '%s', using default %d", val, maxBufferSize)
		}
	}
	// 1. Set default configuration
	initialRegex := regexp.MustCompile(".*") // Match everything by default
	globalConfig.Store(ServerConfig{
		Regex:     initialRegex,
		TargetURL: "", // No target by default (logs to stdout)
	})

	mux := http.NewServeMux()

	// 2. Data Ingest Handler (The "Grep" logic)
	mux.HandleFunc("/ingest", handleIngest)

	// 3. Control Handler (Update Regex/Target)
	mux.HandleFunc("/config", handleConfig)

	// 4. Output Handler (Read buffered matches)
	mux.HandleFunc("/out", handleOut)

	log.Printf("Grep Service starting on port %s", port)
	return http.ListenAndServe(":"+port, mux)
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}

	cfg := globalConfig.Load().(ServerConfig)

	// Use the existing Searcher logic?
	// Searcher is designed for files with callbacks. We can reuse ScanStream.
	searcher := &Searcher{
		Regex:        cfg.Regex,
		ContextLines: 1, // Default context
	}

	// ScanStream writes formatted output to callback.
	// We want to capture it and forward it.

	// Note: We scan the request body directly.
	// We must ensure we don't block too long or use too much memory.
	// ScanStream uses bufio.Scanner which handles streaming.

	searcher.ScanStream(r.Body, "stream", func(res string) {
		// Log to stdout (always handy for docker logs)
		fmt.Println(res)

		// Append to in-memory buffer
		bufferMutex.Lock()
		if len(matchBuffer) >= maxBufferSize {
			// Drop oldest if full (simple ring buffer-ish behavior)
			matchBuffer = matchBuffer[1:]
		}
		matchBuffer = append(matchBuffer, res)
		bufferMutex.Unlock()

		// If target URL is set, forward it
		if cfg.TargetURL != "" {
			go forwardMatch(res, cfg.TargetURL)
		}
	})

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "Processed")
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Use PUT or POST", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params or body? User suggested POST/PUT /config
	// Let's support query params for simplicity as per example, or JSON body.
	// The example used query params: ?regex=...&target=...

	newPattern := r.URL.Query().Get("regex")
	newTarget := r.URL.Query().Get("target")

	if newPattern == "" && newTarget == "" {
		http.Error(w, "Missing regex or target params", http.StatusBadRequest)
		return
	}

	// Security: Length Limit
	if len(newPattern) > 100 {
		http.Error(w, "Regex too long (max 100 chars)", http.StatusBadRequest)
		return
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	currentCfg := globalConfig.Load().(ServerConfig)
	nextCfg := currentCfg

	if newPattern != "" {
		re, err := regexp.Compile(newPattern)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid Regex: %v", err), http.StatusBadRequest)
			return
		}
		nextCfg.Regex = re
	}

	if newTarget != "" {
		nextCfg.TargetURL = newTarget
	}

	globalConfig.Store(nextCfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "updated",
		"regex":  nextCfg.Regex.String(),
		"target": nextCfg.TargetURL,
	})
}

func forwardMatch(line string, url string) {
	// Simple POST forwarder.
	// In a real high-throughput system you'd want a worker pool.
	// For now, we spawn a goroutine per match (beware of limits).
	resp, err := http.Post(url, "text/plain", bytes.NewBufferString(line))
	if err != nil {
		log.Printf("Failed to forward to %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
}

func handleOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET", http.StatusMethodNotAllowed)
		return
	}

	bufferMutex.Lock()
	// Copy buffer to avoid holding lock while writing
	// (or just json encode directly if fast enough, but copy is safer for concurrency)
	matches := make([]string, len(matchBuffer))
	copy(matches, matchBuffer)
	bufferMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}
