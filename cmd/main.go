// Version 1.0.6
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: Apr 19 2026
package main

import (
	"fmt"
	"os"
	"time"

	"cert-checker/internal/checker"
	"cert-checker/internal/config"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"

	"github.com/charmbracelet/huh"
)

func main() {
	var urls []string
	var err error

	// initialize config (checks/creates urls.txt in the background)
	urlsFromConfig, _, err := config.InitConfig()
	if err != nil {
		fmt.Printf("%sConfiguration error: %v%s\n", output.ColRed, err, output.ColReset)
		os.Exit(1)
	}

	// print menu
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

	// determine URLs
	if input == "" {
		// no input > use URLs from config file
		urls = urlsFromConfig
		fmt.Printf("%sUsing %d default URLs from config...%s\n\n", output.ColGreen, len(urls), output.ColReset)
	} else {
		urls, err = parser.ParseInput(input)
		if err != nil || len(urls) == 0 {
			fmt.Printf("%sError: No URLs found (%v)%s\n", output.ColRed, err, output.ColReset)
			os.Exit(1)
		}
	}

	results := make([]checker.CertInfo, len(urls))
	for i, u := range urls {
		results[i] = checker.CheckCertExpiry(u, 5*time.Second)
	}

	// print results
	output.PrintResults(results)

	// JSON export / save
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

	// define filename
	filename := fmt.Sprintf("cert-report-%s.json", time.Now().Format("20060102-150405"))

	// export
	if err := output.ExportJSON(results, filename); err != nil {
		fmt.Printf("%sError saving: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	fmt.Printf("\n%sSaved successfully: %s%s\n", output.ColGreen, filename, output.ColReset)
}
