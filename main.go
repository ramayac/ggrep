package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	version     = 0.2
	allFlag     = "--all"
	outFilename = "out.txt"
)

// App configures the application
type App struct {
	searcher    *Searcher
	ext         string
	isVerbose   bool
	output      io.Writer
	outputFile  *os.File
	matcherName string // Name of the executable to exclude
}

func main() {
	// Parse Flags
	var (
		flagSilent = flag.Bool("s", false, "Silent mode (no console output)")
		flagLines  = flag.Int("lines", 1, "Number of context lines")
		flagExt    = flag.String("ext", "", "File extension filter or -all")
	)

	// Custom usage message to match original vaguely, but more standard
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <regex> [extension]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample: %s \"func\" .go -lines=2\n", os.Args[0])
	}

	flag.Parse()

	// Positional arguments override checks (for backward compatibility where possible,
	// but standardizing on flags is better. The plan said we'd use flags).
	// However, the user might still type `./ggrep regex ext`.
	// Let's try to handle mixed args if possible, or just strict flags.
	// Given the instructions, I'll stick to a clean flag implementation but handle the positional 'regex' and 'ext' if provided purely positionally to be nice.

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	regexStr := args[0]

	// If ext wasn't provided via flag, check if it's the second arg
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

	// Setup Output
	f, err := os.Create(outFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Setup App
	app := &App{
		searcher: &Searcher{
			Regex:        r,
			ContextLines: *flagLines,
		},
		ext:        ext,
		isVerbose:  !*flagSilent,
		output:     f, // We write to file primarily
		outputFile: f,
	}

	// Get self executable name
	if exe, err := os.Executable(); err == nil {
		app.matcherName = filepath.Base(exe)
	}

	// Banner
	app.log("****************************************")
	app.log(fmt.Sprintf("*   Go Grep, version %.1f               *", version))
	app.log("*                                      *")
	app.log("****************************************")
	app.log("")
	app.log(fmt.Sprintf("Starting search for '%s', VERBOSE: %v, lines : %d", regexStr, app.isVerbose, *flagLines))

	startTime := time.Now()

	// Walk
	err = filepath.Walk(".", app.walkFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
	}

	elapsed := time.Since(startTime)
	app.log(fmt.Sprintf("\n\rFinished (%d ms)", elapsed.Milliseconds()))
}

// log writes to both file and stdout (if verbose)
func (app *App) log(msg string) {
	if app.isVerbose {
		fmt.Println(msg)
	}
	// The original used \r\n, preserving that for now
	fmt.Fprintf(app.outputFile, "%s\r\n", msg)
}

func (app *App) walkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	// Filters
	if strings.Contains(path, outFilename) {
		return nil
	}
	if app.matcherName != "" && strings.Contains(path, app.matcherName) {
		return nil
	}

	// Extension check
	shouldProcess := false
	if app.ext == allFlag {
		shouldProcess = true
	} else if strings.Contains(info.Name(), app.ext) {
		shouldProcess = true
	}

	if shouldProcess {
		app.processFile(path)
	}

	return nil
}

func (app *App) processFile(path string) {
	if app.isVerbose {
		fmt.Printf("* Searching in file: '%s' *\r\n", path)
		fmt.Fprintf(app.outputFile, "* Searching in file: '%s' *\r\n", path)
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

	results := app.searcher.ScanStream(f, filepath.Base(path))
	for _, res := range results {
		app.log(res)
	}
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
			fmt.Printf("*** Scanning compressed file: '%s' ***\r\n", f.Name)
			fmt.Fprintf(app.outputFile, "*** Scanning compressed file: '%s' ***\r\n", f.Name)
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}

		results := app.searcher.ScanStream(rc, f.Name)
		for _, res := range results {
			app.log(res)
		}
		rc.Close()
	}
}
