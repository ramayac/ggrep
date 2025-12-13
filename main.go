package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	version      = 0.1
	constAsterix = "-all"
	paramSilent  = "-s"
	usageMessage = "Usage: ./ggrep regex [ext|-all] [lineas] [workers] [-s]"
	outFilename  = "out.txt"
)

// Global flags
var flagBeVerbose = true

// Logger struct
type Logger struct {
	file *os.File
	bw   *bufio.Writer
}

func NewLogger() (*Logger, error) {
	f, err := os.Create(outFilename)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file: f,
		bw:   bufio.NewWriter(f),
	}, nil
}

func (l *Logger) Log(msg string) {
	if flagBeVerbose {
		fmt.Println(msg)
	}
	l.bw.WriteString(msg + "\r\n")
}

func (l *Logger) LogEmpty() {
	if flagBeVerbose {
		fmt.Println()
	}
	l.bw.WriteString("\r\n")
}

func (l *Logger) Close() {
	l.bw.Flush()
	l.file.Close()
}

// Main logic container
type App struct {
	logger       *Logger
	regex        *regexp.Regexp
	ext          string
	contextLines int
	startTime    time.Time
}

func main() {
	logger, err := NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	defer logger.Close()

	logger.Log("****************************************")
	logger.Log("*                                      *")
	logger.Log(fmt.Sprintf("*   Go Grep, version %.1f               *", version))
	logger.Log("*                                      *")
	logger.Log("* Author: @ramayac, 2025               *")
	logger.Log("****************************************")
	logger.LogEmpty()

	app := &App{logger: logger}
	app.run(os.Args[1:])
}

func (app *App) run(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usageMessage)
		return
	}

	var regexStr, ext, silent string
	lines := 1
	workers := runtime.NumCPU()

	// Acceptable forms:
	// 2 args: regex ext
	// 3 args: regex ext lines OR regex ext -s
	// 4 args: regex ext lines workers OR regex ext lines -s
	// 5 args: regex ext lines workers -s

	if len(args) < 2 || len(args) > 5 {
		fmt.Fprintln(os.Stderr, usageMessage)
		return
	}

	regexStr = args[0]
	ext = args[1]

	// Parse optional args
	if len(args) >= 3 {
		// args[2] may be lines or silent
		if args[2] == paramSilent {
			silent = args[2]
		} else if v, err := strconv.Atoi(args[2]); err == nil {
			if v > 0 {
				lines = v
			}
		} else {
			fmt.Fprintln(os.Stderr, "Lines must be a valid number.")
			return
		}
	}

	if len(args) >= 4 {
		// args[3] may be workers (number) or silent
		if args[3] == paramSilent {
			silent = args[3]
		} else if v, err := strconv.Atoi(args[3]); err == nil {
			workers = v
		} else {
			fmt.Fprintln(os.Stderr, "Workers must be a valid number or -s")
			return
		}
	}

	if len(args) == 5 {
		if args[4] == paramSilent {
			silent = args[4]
		} else {
			fmt.Fprintln(os.Stderr, "Unknown parameter: ", args[4])
			return
		}
	}

	// Clamp workers
	if workers < 1 {
		workers = 1
	}
	if workers > runtime.NumCPU() {
		workers = runtime.NumCPU()
	}

	if strings.TrimSpace(regexStr) == "" {
		fmt.Fprintln(os.Stderr, "Missing regular expression or search string")
		return
	}

	if strings.TrimSpace(ext) == "" {
		fmt.Fprintln(os.Stderr, "Missing file extension")
		return
	} else if ext == constAsterix {
		fmt.Fprintln(os.Stderr, "WARNING: All found files will be processed")
	}

	if lines <= 0 {
		fmt.Fprintln(os.Stderr, "Lines must be > 1.")
		lines = 1 // Safety default
	}

	if silent == paramSilent {
		flagBeVerbose = false
	}

	// Compile regex
	r, err := regexp.Compile(regexStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in regex: %v\n", err)
		return
	}
	app.regex = r
	app.ext = ext
	app.contextLines = lines

	// Start search
	currentDir := "."
	app.startTime = time.Now()

	app.logger.Log(fmt.Sprintf("Starting search for '%s', VERBOSE: %v, lines : %d",
		regexStr, flagBeVerbose, lines))

	// Determine executable name for self-exclusion
	exePath, err := os.Executable()
	var exeName string
	if err == nil {
		exeName = filepath.Base(exePath)
	}

	// Walk directory (Recursion)
	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Filter logic
		if strings.Contains(path, outFilename) || (exeName != "" && strings.Contains(path, exeName)) {
			return nil
		}

		shouldProcess := false
		if app.ext == constAsterix {
			shouldProcess = true
		} else if strings.Contains(info.Name(), app.ext) {
			shouldProcess = true
		}

		if shouldProcess {
			app.processFile(path)
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
	}

	elapsed := time.Since(app.startTime)
	app.logger.Log(fmt.Sprintf("\n\rFinished (%d ms)", elapsed.Milliseconds()))
}

func (app *App) processFile(path string) {
	if flagBeVerbose {
		app.logger.Log(fmt.Sprintf("* Searching in file: '%s' *", path))
	}

	// Check if Zip
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		app.readZip(path)
	} else {
		app.readTextFile(path)
	}
}

func (app *App) readTextFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		return
	}
	defer file.Close()

	app.scanStream(file, filepath.Base(path))
}

func (app *App) readZip(path string) {
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

		if flagBeVerbose {
			app.logger.Log(fmt.Sprintf("*** Scanning compressed file: '%s' ***", f.Name))
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		app.scanStream(rc, f.Name)
		rc.Close()
	}
}

// scanStream handles the logic of reading lines and checking matches
func (app *App) scanStream(r io.Reader, filename string) {
	scanner := bufio.NewScanner(r)
	lineNum := 0

	// Create a custom logic to handle the "print N lines after match" requirement
	// Since bufio.Scanner doesn't easily allow peeking ahead, we control the flow manually.

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if app.match(line, filename, lineNum, true) {
			// If match found, and we need to print subsequent lines
			remaining := app.contextLines - 1
			for remaining > 0 && scanner.Scan() {
				// Note: Original code increments line count but prints subsequent lines
				// without file prefix in the secondary loop?
				// The original code calls match(..., false) which formats differently.
				lineNum++
				nextLine := scanner.Text()
				app.match(nextLine, filename, lineNum, false)
				remaining--
			}
		}
	}
}

func (app *App) match(line string, filename string, lineNum int, isSearch bool) bool {
	if line == "" {
		return false
	}

	var output string

	if isSearch {
		// Logic from Grep.search()
		// (Original code had excludeList logic here, but it was initialized empty in Main)
		if app.regex.MatchString(line) {
			output = fmt.Sprintf("    %s:%d:%s", filename, lineNum, line)
		}
	} else {
		// Logic from Lector.match(..., false)
		output = fmt.Sprintf("    %s:%d:%s", filename, lineNum, line)
	}

	if output != "" {
		app.logger.Log(output)
		return true // Match found or forced print
	}
	return false
}
