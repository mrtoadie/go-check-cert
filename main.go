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

const (
	ColReset  = "\033[0m"
	ColRed    = "\033[31m"
	ColGreen  = "\033[32m"
	ColYellow = "\033[33m"
	ColBlue   = "\x1b[34m"
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

// Helper: Farbe basierend auf Status
func getColor(status string) string {
	switch status {
	case "OK": return ColGreen
	case "SOON", "WARNING": return ColYellow
	default: return ColRed
	}
}

// Helper: Tage-Farbe
func getDaysColor(days int) string {
	if days < 30 { return ColRed }
	if days < 60 { return ColYellow }
	return ColGreen
}

// Check certificate
func checkCertificate(url string, timeout time.Duration) CertInfo {
	info := CertInfo{URL: url}
	url = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	if !strings.Contains(url, ":") {
		url += ":443"
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", url, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		info.Error, info.Status = err, "ERROR"
		return info
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		info.Error, info.Status = fmt.Errorf("no certs"), "ERROR"
		return info
	}

	cert := certs[0]
	info.Issuer = cert.Issuer.CommonName
	info.NotBefore, info.NotAfter = cert.NotBefore, cert.NotAfter
	info.DaysRemaining = int(info.NotAfter.Sub(time.Now()).Hours() / 24)

	if info.DaysRemaining < 0 { 
		info.Status = "EXPIRED" 
		}	else if info.DaysRemaining < 30 { info.Status = "WARNING" 
		}	else if info.DaysRemaining < 60 { info.Status = "SOON" 
		}	else { info.Status = "OK" }

	return info
}

// Liest URLs aus einer Datei
func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil { return nil, err }
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

// Parst Eingabe: Entweder Datei (liest alle URLs) ODER kommagetrennte URLs
func parseInput(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" { input = "urls.txt" }

	// Prüfen, ob es eine existierende Datei ist
	if _, err := os.Stat(input); err == nil {
		return readURLsFromFile(input)
	}

	// Sonst als kommagetrennte URLs behandeln
	var urls []string
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			urls = append(urls, p)
		}
	}
	return urls, nil
}

func printResults(results []CertInfo) {
	fmt.Printf("%s=== RESULTS ===%s\n\n", ColBlue, ColReset)

	for i, r := range results {
		num := i + 1
		c := getColor(r.Status)
		daysC := getDaysColor(r.DaysRemaining)

		fmt.Printf("%d. %s%s%s\n", num, c, r.URL, ColReset)
		fmt.Printf("   Days: %s%3d%s | Valid: %s → %s\n", daysC, r.DaysRemaining, ColReset,
			r.NotBefore.Format("02.01.06"), r.NotAfter.Format("02.01.06"))
		fmt.Printf("   Issuer: %s\n", r.Issuer)
		
		if r.Error != nil {
			fmt.Printf("   Error: %s%s%s\n", ColRed, r.Error, ColReset)
		}
		fmt.Printf("%s------------------------------------%s\n", ColBlue, ColReset)
	}

	// Summary Count
	counts := map[string]int{}
	for _, r := range results { counts[r.Status]++ }

	fmt.Printf("%s=== SUMMARY ===%s\n", ColBlue, ColReset)
	fmt.Printf("%sOK: %d%s | %sWarn: %d%s | %sExp: %d%s | %sErr: %d%s\n",
		ColGreen, counts["OK"], ColReset,
		ColYellow, counts["SOON"]+counts["WARNING"], ColReset,
		ColRed, counts["EXPIRED"], ColReset,
		ColRed, counts["ERROR"], ColReset)
}

func main() {
	var input string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("🔍 SSL Checker").
				Description("Dateiname ODER URLs (durch Komma getrennt)").
				Value(&input),
		),
	).Run()

	if err != nil {
		fmt.Printf("%sAbbruch.%s\n", ColRed, ColReset)
		return
	}

	urls, err := parseInput(input)
	if err != nil || len(urls) == 0 {
		fmt.Printf("%sFehler: Keine URLs gefunden (%v)%s\n", ColRed, err, ColReset)
		return
	}

	results := make([]CertInfo, len(urls))
	for i, u := range urls {
		results[i] = checkCertificate(u, 5*time.Second)
	}

	printResults(results)
}