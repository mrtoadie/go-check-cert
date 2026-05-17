// Version 1.1.6
// Autor: 	MrToadie
// GitHub: 	https://github.com/mrtoadie/
// Repo: 		https://github.com/mrtoadie/go-check-cert
// License: MIT
// last modification: May 14 2026
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	//"strconv"

	"cert-checker/internal/checker"
	"cert-checker/internal/config"
	"cert-checker/internal/constants"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"
	"cert-checker/internal/schedule"
	"cert-checker/internal/web"

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

// helper funcs for flag aliase
func addStringAlias(flagPtr *string, shortName, longName, usage string) {
	flag.StringVar(flagPtr, shortName, "", fmt.Sprintf("Alias for --%s", longName))
}

func addBoolAlias(flagPtr *bool, shortName, longName, usage string) {
	flag.BoolVar(flagPtr, shortName, false, fmt.Sprintf("Alias for --%s", longName))
}

func main() {
	// define flag map
	validFlags := map[string]bool{
		"-file": true, "-f": true,
		"-cron": true, "-c": true,
		"-ci-mode": true, "-ci": true,
		"-list": true, "-ls": true,
		"-log": true, "-l": true,
		"-help": true, "-h": true,
		"-web": true, "-w": true,
		"-cert":     true,
		"-key":      true,
		"-download": true, "-dl": true,
	}
	// pre-validation of arguments
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
			if !validFlags[arg] {
				fmt.Println(output.ColRed, "Error: flag provided but not defined:", arg, output.ColReset)
				fmt.Println(output.ColYellow, "Hint: Use -h or -help for usage information.", output.ColReset)
				os.Exit(2)
			}
		}
	}

	// define flags & aliase
	fileFlag := flag.String("file", "", "Path to a local .pem/.crt file")
	addStringAlias(fileFlag, "f", "file", "Path to a local .pem/.crt file")

	cronFlag := flag.Bool("cron", false, "Create & manage cron jobs")
	addBoolAlias(cronFlag, "c", "cron", "Create & manage cron jobs")

	ciModeFlag := flag.Bool("ci-mode", false, "CI/CD Mode: Non-interactive, uses urls.txt automatically")
	addBoolAlias(ciModeFlag, "ci", "ci-mode", "CI/CD Mode: Non-interactive, uses urls.txt automatically")

	listFlag := flag.Bool("list", false, "Show all cron jobs with 'cert-checker'")
	addBoolAlias(listFlag, "ls", "list", "Show all cron jobs with 'cert-checker'")

	logFlag := flag.Bool("log", false, "Show cron job log file")
	addBoolAlias(logFlag, "l", "log", "Show cron job log file")

	helpFlag := flag.Bool("help", false, "Show help")
	addBoolAlias(helpFlag, "h", "help", "Show help")

	webFlag := flag.Bool("web", false, "Start web dashboard on localhost:8080")
	addBoolAlias(webFlag, "w", "web", "Start web dashboard on localhost:8080")

	certFlag := flag.String("cert", "", "Path to SSL certificate file (.pem/.crt)")
	keyFlag := flag.String("key", "", "Path to SSL private key file (.pem)")

	downloadFlag := flag.Bool("download", false, "Download certificate files to certs directory")
	addBoolAlias(downloadFlag, "dl", "download", "Download certificate files to certs directory")

	// usage func
	flag.Usage = func() {
		fmt.Println(output.ColGreen, "\n Examples:", output.ColReset)
		fmt.Println("  cert-checker -file cert.pem")
		fmt.Println("  cert-checker -cron")
		fmt.Println("  cert-checker -log")
		fmt.Println("  cert-checker -web -cert ./certs/cert.pem -key ./certs/key.pem ")
		os.Exit(0)
	}
	flag.Parse()

	// create default ini file if not exists
	config.EnsureDefaults()

	if *cronFlag {
		schedule.ScheduleMain()
		os.Exit(0)
	}

	if *ciModeFlag {
		runCIMode()
		os.Exit(0)
	}

	if *listFlag {
		schedule.ListAndManageJobs()
		os.Exit(0)
	}

	if *logFlag {
		schedule.ViewLogs()
		os.Exit(0)
	}

	if *webFlag {
		web.StartServer("8080", *certFlag, *keyFlag, constants.Version)
	}



	if *helpFlag {
		fmt.Println(output.ColBlue, "\ncert-checker "+constants.Version, output.ColReset)

		fmt.Println(output.ColYellow, "\n Usage: cert-checker [options]", output.ColReset)
		fmt.Println(output.ColBlue, "\n Options:", output.ColReset)

		fmt.Println(output.ColYellow, " -f, -file string", output.ColReset)
		fmt.Println(output.ColBlue, "         Path to a local .pem/.crt file", output.ColReset)

		fmt.Println(output.ColYellow, " -c, -cron", output.ColReset)
		fmt.Println(output.ColBlue, "         Create & manage cron jobs", output.ColReset)

		fmt.Println(output.ColYellow, " -ls, -list", output.ColReset)
		fmt.Println(output.ColBlue, "         Show / remove cron jobs", output.ColReset)

		fmt.Println(output.ColYellow, " -l, -log", output.ColReset)
		fmt.Println(output.ColBlue, "         Show cron job log file", output.ColReset)

		fmt.Println(output.ColYellow, " -ci, -ci-mode", output.ColReset)
		fmt.Println(output.ColBlue, "         CI/CD Mode: Non-interactive, uses urls.txt automatically", output.ColReset)

		fmt.Println(output.ColYellow, " -w, -web", output.ColReset)
		fmt.Println(output.ColBlue, "         Start web dashboard on localhost:8080", output.ColReset)

		fmt.Println(output.ColYellow, " -cert", output.ColReset)
		fmt.Println(output.ColBlue, "         Path to SSL certificate file (.pem/.crt)", output.ColReset)

		fmt.Println(output.ColYellow, " -key", output.ColReset)
		fmt.Println(output.ColBlue, "         Path to SSL private key file (.pem)", output.ColReset)

		fmt.Println(output.ColYellow, " -h, -help", output.ColReset)
		fmt.Println(output.ColBlue, "         Show this help message", output.ColReset)

		flag.Usage()
		os.Exit(0)
	}

	var urls []string
	var inputType InputType
	var err error

	// get local file input
	if *fileFlag != "" {
		// flag mode explicit file
		if _, err := os.Stat(*fileFlag); os.IsNotExist(err) {
			fmt.Printf("%sError: File not found: %s%s\n", output.ColRed, *fileFlag, output.ColReset)
			os.Exit(1)
		}
		urls = []string{*fileFlag}
		inputType = TypeFile
	} else {
		// interactive mode
		urlsFromConfig, _, err := config.InitConfig()
		if err != nil {
			fmt.Printf("%sConfiguration error: %v%s\n", output.ColRed, err, output.ColReset)
			os.Exit(1)
		}
		// huh input form
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
			if checker.IsCertFile(urls[0]) {
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
	timeout := config.GetTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// errgroup
	// limits the parallel goroutines to 10
	// this prevents network stack from being flooded
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	// prepare results (same length as urls)
	results := make([]checker.CertInfo, len(urls))

	timeoutPerRequest := time.Duration(config.GetTimeout()) * time.Second
	// loop URLs
	for i, u := range urls {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var hostname string
			var checkResult checker.CertInfo

			if inputType == TypeFile {
				hostname = ""
				checkResult = checker.CheckCertExpiry(u, hostname, timeoutPerRequest)
			} else {
				hostname = checker.ExtractHostname(u)
				if hostname == "" {
					checkResult = checker.CertInfo{URL: u, Status: "ERROR", Error: fmt.Errorf("empty hostname")}
					results[i] = checkResult
					return nil
				}
				checkResult = checker.CheckCertExpiry(u, hostname, timeoutPerRequest)
			}
			results[i] = checkResult
			return nil
		})
	}

	// wait until all goroutines are finished
	if err := g.Wait(); err != nil {
		fmt.Printf("%sBatch processing interrupted: %v%s\n", output.ColRed, err, output.ColReset)
		// even if a timeout occurred, show the previous results
	}
///////////
fmt.Printf("\n%s=== DEBUG RESULTS ===%s\n", output.ColBlue, output.ColReset)
validCount := 0
nilCertCount := 0
for i, r := range results {
    fmt.Printf(" [%d] %s | Status: %s | RawCert: %v\n", i, r.URL, r.Status, r.RawCert != nil)
    if r.Status == "VALID" {
        validCount++
        if r.RawCert == nil {
            nilCertCount++
        }
    }
}
fmt.Printf("Total Valid: %d, Valid with Nil Cert: %d\n\n", validCount, nilCertCount)

if *downloadFlag {
		//debug
		fmt.Printf("\n%sDownloading valid certificates...%s\n", output.ColBlue, output.ColReset)
	fmt.Printf("BLA")
		certDir, err := config.GetCertPath()
		
		if err != nil {
			fmt.Printf("%sError determining cert directory: %v%s\n", output.ColRed, err, output.ColReset)
		} else {
			// Make sure folder
			if err := os.MkdirAll(certDir, 0755); err != nil {
				fmt.Printf("%sWarning: Could not create cert dir: %v%s\n", output.ColYellow, err, output.ColReset)
			} else {
				fmt.Print("\n\nBLUB", certDir)
				savedCount := 0
				for _, r := range results {
					if r.Status == "VALID" && r.RawCert != nil {
						
						fmt.Printf("dsfsdfdsf")
						hostname := r.URL
						if r.URL != "" && strings.Contains(r.URL, "://") {
							
							u, _ := url.Parse(r.URL)
							if u != nil {
								hostname = u.Hostname()
							}
						}
					fmt.Printf("== Not valid ==")
						if err := checker.SaveCert(r.RawCert, hostname, certDir); err == nil {
							savedCount++
							fmt.Printf("%sSaved:%s %s/%s.pem\n", output.ColGreen, output.ColReset, certDir, hostname)
						} else {
							fmt.Printf("%sError saving %s: %v%s\n", output.ColRed, hostname, err, output.ColReset)
						}
					}
				}
				fmt.Printf("%sDownloaded %d certificate(s).%s\n", output.ColBlue, savedCount, output.ColReset)
			}
		}
	}
/////////

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

	if err := saveReport(results); err != nil {
		fmt.Printf("%sError saving: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}
}

func runCIMode() {
	// load URLs from config file
	urls, _, err := config.InitConfig()
	if err != nil {
		fmt.Printf("%sConfiguration error: %v%s\n", output.ColRed, err, output.ColReset)
		os.Exit(1)
	}

	if len(urls) == 0 {
		fmt.Printf("%sNo URLs found in configuration file.%s\n", output.ColYellow, output.ColReset)
		os.Exit(1)
	}

	fmt.Printf("%sCheck %d URLs from urls.txt...%s\n\n", output.ColBlue, len(urls), output.ColReset)

	timeoutPerRequest := time.Duration(config.GetTimeout()) * time.Second

	// check URLs
	results := make([]checker.CertInfo, len(urls))
	for i, u := range urls {
		hostname := checker.ExtractHostname(u)
		if hostname == "" {
			results[i] = checker.CheckCertExpiry(u, hostname, timeoutPerRequest)
			continue
		}
		results[i] = checker.CheckCertExpiry(u, hostname, timeoutPerRequest)
	}

	// output results (WITHOUT interactive query)
	output.PrintResults(results)

	// save JSON
	if err := saveReport(results); err != nil {
		fmt.Printf("%sError saving: %v%s\n", output.ColRed, err, output.ColReset)
		os.Exit(3)
	}

	// exit code based on results
	os.Exit(checker.CalculateExitCode(results))
}

// saveReport saves the results as JSON
func saveReport(results []checker.CertInfo) error {
	outputDir, err := config.GetOutputPath()
	if err != nil {
		return fmt.Errorf("could not determine output dir: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("cert-report-%s.json", time.Now().Format(constants.ReportDateFormat)))

	// write JSON
	if err := output.ExportJSON(results, filename); err != nil {
		return fmt.Errorf("could not save JSON: %w", err)
	}

	fmt.Printf("\n%sSaved successfully to: %s%s\n", output.ColGreen, filename, output.ColReset)
	return nil
}
