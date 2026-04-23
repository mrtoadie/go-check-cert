// Version 1.0.6
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: Apr 19 2026
package main

import (
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

	// perform check
	results := make([]checker.CertInfo, len(urls))
	for i, u := range urls {
		var hostname string

		if inputType == TypeFile {
			hostname = ""
			fmt.Printf("%sChecking local file: %s%s\n", output.ColBlue, u, output.ColReset)
		} else {
			// use centralized hostname extraction com checker.go
			hostname = checker.ExtractHostname(u)

			if hostname == "" {
				fmt.Printf("%sWarning: Empty hostname for '%s', skipping...%s\n", output.ColYellow, u, output.ColReset)
				results[i] = checker.CertInfo{URL: u, Status: "ERROR", Error: fmt.Errorf("empty hostname")}
				continue
			}

			fmt.Printf("%sChecking remote: %s (Host: %s)%s\n", output.ColBlue, u, hostname, output.ColReset)
		}

		results[i] = checker.CheckCertExpiry(u, hostname, 5*time.Second)
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

	homeDir, _ := os.UserHomeDir()
	// Note: Ensure config.ConfigDir is accessible or define it here if needed
	// Assuming config.ConfigDir is exported or accessible
	configDir := config.ConfigDir
	filename := filepath.Join(homeDir+"/"+configDir, fmt.Sprintf("cert-report-%s.json", time.Now().Format("20060102-150405")))

	if err := output.ExportJSON(results, filename); err != nil {
		fmt.Printf("%sError saving: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	fmt.Printf("\n%sSaved successfully to: %s%s\n", output.ColGreen, filename, output.ColReset)
}
