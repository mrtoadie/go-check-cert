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
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
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

// --- Zertifikat prüfen ---
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

// --- URLs aus Datei lesen ---
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

// --- Ergebnisse anzeigen ---
func printResults(results []CertInfo) {
	fmt.Printf("\n%s=== ERGEBNISSE ===%s\n\n", Bold, Reset)
	
	for _, r := range results {
		var color, emoji string
		switch r.Status {
		case "OK":
			color, emoji = Green, "✅"
		case "SOON":
			color, emoji = Yellow, "⚠️"
		case "WARNING":
			color, emoji = Red, "🔴"
		case "EXPIRED":
			color, emoji = Red, "💥"
		default:
			color, emoji = Red, "🔧"
		}
		
		fmt.Printf("%s%s %s%s %s\n", Bold, color, r.URL, Reset, emoji)
		fmt.Printf("   Issuer: %s\n", r.Issuer)
		fmt.Printf("   Valid:  %s → %s\n", r.NotBefore.Format("2006-01-02"), r.NotAfter.Format("2006-01-02"))
		
		daysColor := Green
		if r.DaysRemaining < 30 {
			daysColor = Red
		} else if r.DaysRemaining < 60 {
			daysColor = Yellow
		}
		fmt.Printf("   Days:   %s%d%s\n", daysColor, r.DaysRemaining, Reset)
		
		if r.Error != nil {
			fmt.Printf("   Error:  %s%s%s\n", Red, r.Error, Reset)
		}
		fmt.Println()
	}
	
	// Zusammenfassung
	ok, warn, exp, err := 0, 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "OK": ok++
		case "SOON", "WARNING": warn++
		case "EXPIRED": exp++
		case "ERROR": err++
		}
	}
	
	fmt.Printf("%s=== SUMMARY ===%s\n", Bold, Reset)
	fmt.Printf("%sOK: %d%s | %sWarn: %d%s | %sExp: %d%s | %sErr: %d%s\n", 
		Green, ok, Reset, Yellow, warn, Reset, Red, exp, Reset, Red, err, Reset)
}

// --- Hauptprogramm ---
func main() {
	var input string
	
	// huh Eingabe
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("🔍 SSL Certificate Checker").
				Description("Dateiname oder URL eingeben (Leer = urls.txt)").
				Value(&input),
		),
	).Run()
	
	if err != nil {
		fmt.Printf("%sAbbruch.%s\n", Red, Reset)
		return
	}
	
	// Datei oder URL bestimmen
	filename := input
	if filename == "" {
		filename = "urls.txt"
	}
	
	var urls []string
	if _, err := os.Stat(filename); err == nil {
		// Datei existiert
		urls, err = readURLsFromFile(filename)
		if err != nil {
			fmt.Printf("%sFehler beim Lesen: %s%s\n", Red, err, Reset)
			return
		}
	} else {
		// Als URL behandeln
		urls = []string{input}
	}
	
	if len(urls) == 0 {
		fmt.Printf("%sKeine URLs gefunden.%s\n", Yellow, Reset)
		return
	}
	
	// Prüfen
	timeout := 5 * time.Second
	results := make([]CertInfo, len(urls))
	
	for i, url := range urls {
		results[i] = checkCertificate(url, timeout)
	}
	
	// Anzeigen
	printResults(results)
	
	// Bestätigung zum Beenden
	var confirm bool
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Beenden?").
				Value(&confirm),
		),
	).Run()
}