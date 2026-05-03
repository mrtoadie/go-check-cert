// Version 1.1.2
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: May 03 2026
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cert-checker/internal/checker"
	"cert-checker/internal/config"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"
	"cert-checker/internal/schedule"

	"github.com/charmbracelet/huh"
	"golang.org/x/sync/errgroup"
)

type InputType int

const (
	TypeEmpty InputType = iota
	TypeFile
	TypeURL
	TypeMixed
	Version = "1.1.2"
)

func main() {
	localFile := flag.String("file", "", "Path to a local .pem/.crt file")
	flag.StringVar(localFile, "f", "", "Path to a local .pem/.crt file (alias)")

	intitSchedule := flag.Bool("c", false, "Cron-Setup")
	flag.BoolVar(intitSchedule, "cron", false, "Cron-Setup (alias)")

	ciMode := flag.Bool("ci", false, "CI/CD Mode: Non-interactive, uses urls.txt automatically")
	flag.BoolVar(ciMode, "ci-mode", false, "CI/CD Mode (alias)")

	listFlag := flag.Bool("list", false, "Show all cron jobs with 'cert-checker'")
	flag.BoolVar(listFlag, "ls", false, "Show cron jobs (alias)")

	showHelp := flag.Bool("h", false, "Show help")

	flag.Parse()

	if *intitSchedule {
		schedule.ScheduleMain()
		os.Exit(0)
	}

	if *ciMode {
		runCIMode()
		os.Exit(0)
	}

	if *listFlag {
		schedule.ListAndManageJobs()
		os.Exit(0)
	}

	if *showHelp {
		fmt.Fprintf(os.Stderr, "cert-checker v%s\n\n", Version)

		// short description
		fmt.Fprintln(os.Stderr, "Usage: cert-checker [options]")
		fmt.Fprintln(os.Stderr, "\nOptions:")

		fmt.Fprintln(os.Stderr, "  -c, -cron")
		fmt.Fprintln(os.Stderr, "        Setup cron jobs")

		fmt.Fprintln(os.Stderr, "  -ls, -list")
		fmt.Fprintln(os.Stderr, "        Show / remove cron jobs")

		fmt.Fprintln(os.Stderr, "  -ci, -ci-mode")
		fmt.Fprintln(os.Stderr, "        CI/CD Mode: Non-interactive, uses urls.txt automatically")

		fmt.Fprintln(os.Stderr, "  -f, -file string")
		fmt.Fprintln(os.Stderr, "        Path to a local .pem/.crt file")

		fmt.Fprintln(os.Stderr, "  -h, -help")
		fmt.Fprintln(os.Stderr, "        Show this help message")
		os.Exit(0)
	}

	var urls []string
	var inputType InputType
	var err error

	// get input
	if *localFile != "" {
		// flag mode: explicit file
		if _, err := os.Stat(*localFile); os.IsNotExist(err) {
			fmt.Printf("%sError: File not found: %s%s\n", output.ColRed, *localFile, output.ColReset)
			os.Exit(1)
		}
		urls = []string{*localFile}
		inputType = TypeFile
	} else {
		// interactive mode
		urlsFromConfig, _, err := config.InitConfig()
		if err != nil {
			fmt.Printf("%sConfiguration error: %v%s\n", output.ColRed, err, output.ColReset)
			os.Exit(1)
		}

		var input string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("=== CERT-CHECKER ===").
					Description("Enter URLs, Filename, or press Enter for defaults").
					Value(&input),
			).WithTheme(huh.ThemeBase16()),
		).Run()

		if err != nil {
			fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
			return
		}

		// parse input
		if input == "" {
			// empty > config file
			urls = urlsFromConfig
			inputType = TypeURL
			fmt.Printf("%sUsing %d default URLs from config...%s\n\n", output.ColGreen, len(urls), output.ColReset)
		} else {
			// not empty > parse and determine type
			urls, err = parser.ParseInput(input)
			if err != nil || len(urls) == 0 {
				fmt.Printf("%sError: No URLs found (%v)%s\n", output.ColRed, err, output.ColReset)
				os.Exit(1)
			}

			// determine type based on the first item using centralized logic
			if checker.IsFilePath(urls[0]) && (strings.HasSuffix(urls[0], ".pem") || strings.HasSuffix(urls[0], ".crt") || strings.HasSuffix(urls[0], ".cer") || strings.HasSuffix(urls[0], ".key")) {
				inputType = TypeFile
				fmt.Printf("%sDetected: Local certificate file%s\n\n", output.ColBlue, output.ColReset)
			} else {
				inputType = TypeURL
				fmt.Printf("%sDetected: Remote URL(s)%s\n\n", output.ColGreen, output.ColReset)
			}
		}
	}
	// concurrency
	// global timeout (e.g. 60 seconds for the entire batch)
	// this prevents from hanging forever if a server doesn't respond
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// errgroup
	// limits the parallel goroutines to 10
	// this prevents network stack from being flooded
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	// prepare results (same length as urls)
	results := make([]checker.CertInfo, len(urls))

	// loop URLs
	for i, u := range urls {
		// catch i and u in a closure so that they are bound correctly in the goroutine
		i, u := i, u

		g.Go(func() error {
			// check whether the context has expired (timeout)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var hostname string
			var checkResult checker.CertInfo

			if inputType == TypeFile {
				hostname = ""

				checkResult = checker.CheckCertExpiry(u, hostname, 5*time.Second)
			} else {
				hostname = checker.ExtractHostname(u)
				if hostname == "" {
					checkResult = checker.CertInfo{URL: u, Status: "ERROR", Error: fmt.Errorf("empty hostname")}
					results[i] = checkResult
					return nil
				}
				// timeout of 5s is per request
				checkResult = checker.CheckCertExpiry(u, hostname, 5*time.Second)
			}
			// save result in the correct position
			results[i] = checkResult
			return nil
		})
	}

	// wait until all goroutines are finished
	if err := g.Wait(); err != nil {
		fmt.Printf("%sBatch processing interrupted: %v%s\n", output.ColRed, err, output.ColReset)
		// even if a timeout occurred, show the previous results
	}

	// print results
	output.PrintResults(results)

	// save JSON
	var saveJSON bool
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save results?").
				Description("Do you want to save the results as JSON?").
				Value(&saveJSON),
		).WithTheme(huh.ThemeBase16()),
	).Run()

	if err != nil || !saveJSON {
		fmt.Println("\nBye...")
		return
	}

	// determine home dir if not exit
	homeDir, _ := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sError: Could not determine home directory: %v%s\n", output.ColRed, err, output.ColReset)
		os.Exit(1)
	}

	// config path/file and filename format
	configDir := config.ConfigDir
	filename := filepath.Join(homeDir, configDir, fmt.Sprintf("cert-report-%s.json", time.Now().Format("20060102-150405")))

	// directory exists before writing?
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		fmt.Printf("%sError creating directory: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	if err := output.ExportJSON(results, filename); err != nil {
		fmt.Printf("%sError saving: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	fmt.Printf("\n%sSaved successfully to: %s%s\n", output.ColGreen, filename, output.ColReset)
}

// NEUE FUNKTION: CI Mode
func runCIMode() {
	// 1. URLs aus Config-Datei laden
	urls, _, err := config.InitConfig()
	if err != nil {
		fmt.Printf("%sKonfigurationsfehler: %v%s\n", output.ColRed, err, output.ColReset)
		os.Exit(1)
	}

	if len(urls) == 0 {
		fmt.Printf("%sKeine URLs in der Konfigurationsdatei gefunden.%s\n", output.ColYellow, output.ColReset)
		os.Exit(1)
	}

	fmt.Printf("%sPrüfe %d URLs aus urls.txt...%s\n\n", output.ColBlue, len(urls), output.ColReset)

	// 2. URLs prüfen (parallele Logik aus main.go übernehmen)
	results := make([]checker.CertInfo, len(urls))
	for i, u := range urls {
		hostname := checker.ExtractHostname(u)
		if hostname == "" {
			results[i] = checker.CertInfo{URL: u, Status: "ERROR", Error: fmt.Errorf("empty hostname")}
			continue
		}
		results[i] = checker.CheckCertExpiry(u, hostname, 5*time.Second)
	}

	// 3. Ergebnisse ausgeben (OHNE interaktive Abfrage)
	output.PrintResults(results)

	// 4. JSON automatisch speichern (optional, ohne Abfrage)
	homeDir, _ := os.UserHomeDir()
	configDir := config.ConfigDir
	filename := filepath.Join(homeDir, configDir, fmt.Sprintf("cert-report-%s.json", time.Now().Format("20060102-150405")))

	if err := output.ExportJSON(results, filename); err != nil {
		fmt.Printf("%sFehler beim Speichern: %v%s\n", output.ColRed, err, output.ColReset)
	} else {
		fmt.Printf("\n%sErgebnisse gespeichert: %s%s\n", output.ColGreen, filename, output.ColReset)
	}

	// 5. Exit Code basierend auf Ergebnissen
	exitCode := 0
	for _, r := range results {
		if r.Status == "EXPIRED" || r.Status == "ERROR" {
			exitCode = 2
			break
		} else if r.Status == "WARNING" || r.Status == "SOON" {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}
