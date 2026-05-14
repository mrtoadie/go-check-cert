// internal/output/output.go
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"cert-checker/internal/checker"
	//"cert-checker/internal/constants"
)

const (
	ColReset  = "\033[0m"
	ColRed    = "\033[31m"
	ColGreen  = "\033[32m"
	ColYellow = "\033[33m"
	ColBlue   = "\033[34m"
)

// ReportData ist die Struktur für JSON-Export (mit korrektem Error-Handling)
type ReportData struct {
	GeneratedAt string       `json:"generated_at"`
	TotalCount  int          `json:"total_count"`
	Results     []CertResult `json:"results"`
}

// CertResult ist eine angepasste Version von CertInfo für JSON
type CertResult struct {
	URL                string   `json:"URL"`
	Issuer             string   `json:"Issuer"`
	Subject            string   `json:"Subject"`
	SerialNumber       string   `json:"SerialNumber"`
	NotBefore          string   `json:"NotBefore"`
	NotAfter           string   `json:"NotAfter"`
	DaysRemaining      int      `json:"DaysRemaining"`
	Status             string   `json:"Status"`
	Error              string   `json:"Error"` // Als String statt error-Objekt!
	KeyAlgorithm       string   `json:"KeyAlgorithm"`
	KeySize            int      `json:"KeySize"`
	SignatureAlgorithm string   `json:"SignatureAlgorithm"`
	SANs               []string `json:"SANs"`
	ChainLength        int      `json:"ChainLength"`
	IsChainComplete    bool     `json:"IsChainComplete"`
	ChainError         string   `json:"ChainError"`
	IsSelfSigned       bool     `json:"IsSelfSigned"`
	RootIssuer         string   `json:"RootIssuer"`
}

// saves the results as JSON
func ExportJSON(results []checker.CertInfo, filename string) error {
	if filename == "" {
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Konvertiere CertInfo zu CertResult (Error als String)
	certResults := make([]CertResult, len(results))
	for i, r := range results {
		certResults[i] = CertResult{
			URL:                r.URL,
			Issuer:             r.Issuer,
			Subject:            r.Subject,
			SerialNumber:       r.SerialNumber,
			NotBefore:          r.NotBefore.Format(time.RFC3339),
			NotAfter:           r.NotAfter.Format(time.RFC3339),
			DaysRemaining:      r.DaysRemaining,
			Status:             r.Status,
			Error:              "", // Error als String
			KeyAlgorithm:       r.KeyAlgorithm,
			KeySize:            r.KeySize,
			SignatureAlgorithm: r.SignatureAlgorithm,
			SANs:               r.SANs,
			ChainLength:        r.ChainLength,
			IsChainComplete:    r.IsChainComplete,
			ChainError:         r.ChainError,
			IsSelfSigned:       r.IsSelfSigned,
			RootIssuer:         r.RootIssuer,
		}
		// Error-Feld sicher setzen
		if r.Error != nil {
			certResults[i].Error = r.Error.Error()
		}
	}

	report := ReportData{
		GeneratedAt: time.Now().Format(time.RFC3339),
		TotalCount:  len(results),
		Results:     certResults,
	}

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("couldn't write JSON: %w", err)
	}

	return nil
}

// selects the color based on the status of the certificate
func GetColor(status string) string {
	switch status {
	case "OK", "VALID":
		return ColGreen
	case "SOON", "WARNING":
		return ColYellow
	default:
		return ColRed
	}
}

// selects the color based on the remaining days
func getDaysColor(days int) string {
	if days < 30 {
		return ColRed
	}
	if days < 60 {
		return ColYellow
	}
	return ColGreen
}

// format and output the results
func PrintResults(results []checker.CertInfo) {
	fmt.Printf("%s=== RESULTS ===%s\n\n", ColBlue, ColReset)

	for i, r := range results {
		num := i + 1
		c := GetColor(r.Status)
		daysC := getDaysColor(r.DaysRemaining)

		fmt.Printf(" %d. %s%s%s\n", num, c, r.URL, ColReset)

		statusLine := fmt.Sprintf("Status: %s%-10s%s", c, r.Status, ColReset)
		if !r.IsChainComplete {
			statusLine += fmt.Sprintf(" | %s CHAIN ISSUE%s", ColRed, ColReset)
		} else {
			statusLine += fmt.Sprintf(" | %s CHAIN OK%s", ColGreen, ColReset)
		}
		fmt.Println(statusLine)

		// chain details
		fmt.Printf("   Chain Length: %d Certificates\n", r.ChainLength)
		if r.IsSelfSigned {
			fmt.Printf("   %sSelf-Signed certificate%s\n", ColYellow, ColReset)
		}

		if !r.IsChainComplete && r.ChainError != "" {
			fmt.Printf("   %sError: %s%s\n", ColRed, r.ChainError, ColReset)
		}

		if r.RootIssuer != "" {
			fmt.Printf("   Root Issuer: %s\n", r.RootIssuer)
		}

		fmt.Printf("   Days: %s%3d%s | Valid: %s → %s\n", daysC, r.DaysRemaining, ColReset,
			r.NotBefore.Format("02. Jan 2006"), r.NotAfter.Format("02. Jan 2006"))
		fmt.Printf("   Issuer: %s\n", r.Issuer)
		fmt.Printf("   Serial Number: %s\n", r.SerialNumber)

		// key info
		fmt.Printf("   Key: %s %d-bit | Sig: %s\n",
			r.KeyAlgorithm, r.KeySize, r.SignatureAlgorithm)

		// sans
		if len(r.SANs) > 0 {
			sansStr := strings.Join(r.SANs, ", ")
			if len(sansStr) > 60 {
				sansStr = sansStr[:57] + "..."
			}
			fmt.Printf("   SANs: %s\n", sansStr)
		}

		if r.Error != nil {
			fmt.Printf("   Error: %s%s%s\n", ColRed, r.Error, ColReset)
		}
		fmt.Printf("%s ------------------------------------%s\n", ColBlue, ColReset)
	}

	// count statuses
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}

	// helper for safe count lookup
	count := func(status string) int {
		return counts[status]
	}

	fmt.Printf("%sValid: %d%s | %sWarn: %d%s | %sExp: %d%s | %sErr: %d%s\n",
		ColGreen, count("VALID"), ColReset,
		ColYellow, count("WARNING"), ColReset,
		ColRed, count("EXPIRED"), ColReset,
		ColRed, count("ERROR"), ColReset)
}

// TruncateString shortens a string and adds "..."
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
