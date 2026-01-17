package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	version = 0.3
	allFlag = "--all"
)

// App configures the application
type App struct {
	searcher    *Searcher
	ext         string
	isVerbose   bool
	output      io.Writer
	matcherName string // Name of the executable to exclude
	wg          sync.WaitGroup
	filesChan   chan string
	mu          sync.Mutex // For synchronized output
}

func main() {
	// Parse Flags
	var (
		flagSilent = flag.Bool("silent", false, "Silent mode (no console output)")
		flagLines  = flag.Int("lines", 1, "Number of context lines")
		flagExt    = flag.String("ext", "", "File extension filter or --all")

		flagVersion = flag.Bool("version", false, "Print version and exit")
		flagServer  = flag.Bool("server", false, "Run in server mode")
		flagPort    = flag.String("port", "8080", "Port to run server on")
	)

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <regex> [extension]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample: %s \"func\" .go --lines=2\n", os.Args[0])
	}

	flag.Parse()

	// Handle Version
	if *flagVersion {
		printBanner()
		return
	}

	// Handle Server Mode
	if *flagServer {
		if err := startServer(*flagPort); err != nil {
			fmt.Fprintf(os.Stderr, "Server closed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	regexStr := args[0]

	ext := *flagExt
	if ext == "" && len(args) > 1 {
		ext = args[1]
	}
	if ext == "" {
		fmt.Fprintln(os.Stderr, "Missing file extension. Use -ext or provide it as second argument.")
		os.Exit(1)
	}

	// Validate Regexp
	r, err := regexp.Compile(regexStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in regex: %v\n", err)
		os.Exit(1)
	}

	// Setup App
	app := &App{
		searcher: &Searcher{
			Regex:        r,
			ContextLines: *flagLines,
		},
		ext:       ext,
		isVerbose: !*flagSilent,
		output:    os.Stdout,
		filesChan: make(chan string, 100),
	}

	// Get self executable name
	if exe, err := os.Executable(); err == nil {
		app.matcherName = filepath.Base(exe)
	}

	// Only log start message if verbose
	app.log(fmt.Sprintf("Starting search for '%s', VERBOSE: %v, lines : %d, workers: %d", regexStr, app.isVerbose, *flagLines, runtime.NumCPU()))

	startTime := time.Now()

	// Start workers
	for i := 0; i < runtime.NumCPU(); i++ {
		app.wg.Add(1)
		go app.worker()
	}

	// Walk
	err = filepath.Walk(".", app.walkFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
	}

	close(app.filesChan)
	app.wg.Wait()

	elapsed := time.Since(startTime)
	app.log(fmt.Sprintf("\nFinished (%d ms)", elapsed.Milliseconds()))
}

func printBanner() {
	fmt.Println("***************************************************")
	fmt.Printf("* GoGrep, version %.1f                             *\n", version)
	fmt.Println("* Toda la vida es sueño, y los sueños, sueños son. *)                                   *")
	fmt.Println("***************************************************")
}

func (app *App) worker() {
	defer app.wg.Done()
	for path := range app.filesChan {
		app.processFile(path)
	}
}

func (app *App) log(msg string) {
	if app.isVerbose {
		app.mu.Lock()
		fmt.Fprintln(app.output, msg)
		app.mu.Unlock()
	}
}

func (app *App) walkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	// Filters
	if app.matcherName != "" && strings.Contains(path, app.matcherName) {
		return nil
	}

	// Extension check
	shouldProcess := false
	if app.ext == allFlag {
		shouldProcess = true
	} else if strings.HasSuffix(info.Name(), app.ext) {
		shouldProcess = true
	}

	if shouldProcess {
		app.filesChan <- path
	}

	return nil
}

func (app *App) processFile(path string) {
	if app.isVerbose {
		app.log(fmt.Sprintf("* Searching in file: '%s' *", path))
	}

	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		app.processZip(path)
	} else {
		app.processText(path)
	}
}

func (app *App) processText(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", path, err)
		return
	}
	defer f.Close()

	app.searcher.ScanStream(f, filepath.Base(path), func(res string) {
		app.log(res)
	})
}

func (app *App) processZip(path string) {
	r, err := zip.OpenReader(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening zip %s: %v\n", path, err)
		return
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if app.isVerbose {
			app.log(fmt.Sprintf("*** Scanning compressed file: '%s' ***", f.Name))
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}

		app.searcher.ScanStream(rc, f.Name, func(res string) {
			app.log(res)
		})
		rc.Close()
	}
}
