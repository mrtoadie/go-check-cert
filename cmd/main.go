// Version 1.1.0
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: Apr 28 2026
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

	"github.com/charmbracelet/huh"
	"golang.org/x/sync/errgroup"
)

type InputType int

const (
	TypeEmpty InputType = iota
	TypeFile
	TypeURL
	TypeMixed
)

func main() {
	localFile := flag.String("file", "", "Path to a local .pem/.crt file")
	flag.Parse()

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
					Title("=== SSL CHECKER ===").
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

			// Ergebnis an der korrekten Position speichern
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
