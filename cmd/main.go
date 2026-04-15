// Version 1.0.3
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: Apr 15 2026
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"cert-checker/internal/checker"
	"cert-checker/internal/config"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"
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
}