// Version 1.0
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: Apr 12 2026
package main

import (
	"fmt"
	"os"
	"time"

	"cert-checker/internal/checker"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"

	"github.com/charmbracelet/huh"
)

func main() {
	var input string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("=== SSL CHECKER ===").
				Description("Filename OR URLs (separated by commas)").
				Value(&input),
		).WithTheme(huh.ThemeBase16()),
	).Run()

	if err != nil {
		fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
		return
	}

	urls, err := parser.ParseInput(input)
	if err != nil || len(urls) == 0 {
		fmt.Printf("%sError: No URLs found (%v)%s\n", output.ColRed, err, output.ColReset)
		os.Exit(1)
	}

	results := make([]checker.CertInfo, len(urls))
	for i, u := range urls {
		results[i] = checker.CheckCertificate(u, 5*time.Second)
	}

	output.PrintResults(results)
}
