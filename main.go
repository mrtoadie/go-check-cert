package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

// Farben
const (
	ColReset  = "\033[0m"
	ColBold   = "\033[1m"
	ColRed    = "\033[31m"
	ColGreen  = "\033[32m"
	ColYellow = "\033[33m"
	ColCyan   = "\033[36m"
)

type CertInfo struct {
	URL           string
	Issuer        string
	NotBefore     time.Time
	NotAfter      time.Time
	DaysRemaining int
	Status        string
	Error         error
}

// Check certificate
func checkCertificate(url string, timeout time.Duration) CertInfo {
	info := CertInfo{URL: url}
	
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if !strings.Contains(url, ":") {
		url = url + ":443"
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", url, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		info.Error = err
		info.Status = "ERROR"
		return info
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		info.Error = fmt.Errorf("no certs")
		info.Status = "ERROR"
		return info
	}

	cert := certs[0]
	info.Issuer = cert.Issuer.CommonName
	info.NotBefore = cert.NotBefore
	info.NotAfter = cert.NotAfter
	info.DaysRemaining = int(info.NotAfter.Sub(time.Now()).Hours() / 24)

	if info.DaysRemaining < 0 {
		info.Status = "EXPIRED"
	} else if info.DaysRemaining < 30 {
		info.Status = "WARNING"
	} else if info.DaysRemaining < 60 {
		info.Status = "SOON"
	} else {
		info.Status = "OK"
	}
	return info
}

// Read URLs from file
func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}

// Show results
func printResults(results []CertInfo) {
	fmt.Printf("\n%s=== RESULTS ===%s\n\n", ColBold, ColReset)
	
	for _, r := range results {
		var color string
		switch r.Status {
		case "OK":
			color = ColGreen
		case "SOON":
			color = ColYellow
		case "WARNING":
			color = ColRed
		case "EXPIRED":
			color = ColRed
		default:
			color = ColRed
		}
		
		fmt.Printf("%s%s %s%s\n", ColBold, color, r.URL, ColReset)
		fmt.Printf("   Issuer: %s\n", r.Issuer)
		fmt.Printf("   Valid:  %s → %s\n", r.NotBefore.Format("2006-01-02"), r.NotAfter.Format("2006-01-02"))
		
		daysColor := ColGreen
		if r.DaysRemaining < 30 {
			daysColor = ColRed
		} else if r.DaysRemaining < 60 {
			daysColor = ColYellow
		}
		fmt.Printf("   Days:   %s%d%s\n", daysColor, r.DaysRemaining, ColReset)
		fmt.Printf("------------------------------------")
		if r.Error != nil {
			fmt.Printf("   Error:  %s%s%s\n", ColRed, r.Error, ColReset)
		}
		fmt.Println()
	}
	
	// Summary
	ok, warn, exp, err := 0, 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "OK": ok++
		case "SOON", "WARNING": warn++
		case "EXPIRED": exp++
		case "ERROR": err++
		}
	}
	
	fmt.Printf("%s=== SUMMARY ===%s\n", ColBold, ColReset)
	fmt.Printf("%sOK: %d%s | %sWarn: %d%s | %sExp: %d%s | %sErr: %d%s\n", 
		ColGreen, ok, ColReset, ColYellow, warn, ColReset, ColRed, exp, ColReset, ColRed, err, ColReset)
}

// main
func main() {
	var input string
	
	// input
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("🔍 SSL Certificate Checker").
				Description("Enter file name or URL (Empty = urls.txt)").
				Value(&input),
		),
	).Run()
	
	if err != nil {
		fmt.Printf("%sAbort.%s\n", ColRed, ColReset)
		return
	}
	
	// specify if the input is a file or URL
	// file must be in the same dir
	filename := input
	if filename == "" {
		filename = "urls.txt"
	}
	
	var urls []string
	if _, err := os.Stat(filename); err == nil {
		// file exists
		urls, err = readURLsFromFile(filename)
		if err != nil {
			fmt.Printf("%sError reading: %s%s\n", ColRed, err, ColReset)
			return
		}
	} else {
		// treat as URL
		urls = []string{input}
	}
	
	if len(urls) == 0 {
		fmt.Printf("%sNo URLs found.%s\n", ColYellow, ColReset)
		return
	}
	
	// check
	timeout := 5 * time.Second
	results := make([]CertInfo, len(urls))
	
	for i, url := range urls {
		results[i] = checkCertificate(url, timeout)
	}
	
	// print the results
	printResults(results)
}