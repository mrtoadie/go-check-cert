// internal/output/output.go
package output

import (
	"bufio"
	"cert-checker/internal/checker"
	"cert-checker/internal/constants"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	ColReset  = "\033[0m"
	ColRed    = "\033[31m"
	ColGreen  = "\033[32m"
	ColYellow = "\033[33m"
	ColBlue   = "\033[34m"
)

// ReportData is the structure for JSON export
type ReportData struct {
	GeneratedAt string       `json:"generated_at"`
	TotalCount  int          `json:"total_count"`
	Results     []CertResult `json:"results"`
}

// CertResult is a customized version of CertInfo for JSON
type CertResult struct {
	URL                string   `json:"URL"`
	Issuer             string   `json:"Issuer"`
	Subject            string   `json:"Subject"`
	SerialNumber       string   `json:"SerialNumber"`
	NotBefore          string   `json:"NotBefore"`
	NotAfter           string   `json:"NotAfter"`
	DaysRemaining      int      `json:"DaysRemaining"`
	Status             string   `json:"Status"`
	Error              string   `json:"Error"`
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

// ExportJSON writes the results as JSON to a file as an indented JSON report
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

	// convert CertInfo to CertResult (Error as String)
	certResults := make([]CertResult, len(results))
	for i, r := range results {
		certResults[i] = CertResult{
			URL:                r.URL,
			Issuer:             r.Issuer,
			Subject:            r.Subject,
			SerialNumber:       r.SerialNumber,
			NotBefore:          r.NotBefore.Format(constants.RFC3339Format),
			NotAfter:           r.NotAfter.Format(constants.RFC3339Format),
			DaysRemaining:      r.DaysRemaining,
			Status:             r.Status,
			Error:              "",
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
		if r.Error != nil {
			certResults[i].Error = r.Error.Error()
		}
	}

	report := ReportData{
		GeneratedAt: time.Now().Format(constants.RFC3339Format),
		TotalCount:  len(results),
		Results:     certResults,
	}

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("couldn't write JSON: %w", err)
	}
	return nil
}

// GetColor returns the ANSI color code for the given status
func GetColor(status string) string {
	switch status {
	case "VALID":
		return ColGreen
	case "SOON", "WARNING":
		return ColYellow
	default:
		return ColRed
	}
}

// getDaysColor returns the ANSI escape code for the given days-remaining value
func getDaysColor(days int) string {
	if days < 30 {
		return ColRed
	}
	if days < 60 {
		return ColYellow
	}
	return ColGreen
}

// printSummary prints the status count footer shared by all result views.
func printSummary(results []checker.CertInfo) {
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	count := func(status string) int { return counts[status] }

	fmt.Printf(" %sValid: %d%s | %sWarn: %d%s | %sExp: %d%s | %sErr: %d%s\n",
		ColGreen, count("VALID"), ColReset,
		ColYellow, count("WARNING"), ColReset,
		ColRed, count("EXPIRED"), ColReset,
		ColRed, count("ERROR"), ColReset)
	fmt.Printf("%s----------------------------------------%s\n", ColBlue, ColReset)
}

// PrintResults prints a compact one-line-per-cert summary to stdout
func PrintResults(results []checker.CertInfo) {
	fmt.Printf("%s=== RESULTS ===%s\n\n", ColBlue, ColReset)

	for i, r := range results {
		num := i + 1
		c := GetColor(r.Status)
		daysC := getDaysColor(r.DaysRemaining)
		fmt.Printf(" %d. %s%s%s\n", num, c, r.URL, ColReset)
		fmt.Printf(" Status: %s%-7s%s | Days:%s%6d%s\n", c, r.Status, ColReset, daysC, r.DaysRemaining, ColReset)
		if r.Error != nil {
			fmt.Printf("   Error: %s%s%s\n", ColRed, r.Error, ColReset)
		}
		fmt.Printf("%s----------------------------------------%s\n", ColBlue, ColReset)
	}
	printSummary(results)
}

// PrintAdvancedResults prints detailed certificate information to stdout
func PrintAdvancedResults(results []checker.CertInfo) {
	fmt.Printf("%s=== RESULTS ===%s\n\n", ColBlue, ColReset)

	for i, r := range results {
		num := i + 1
		c := GetColor(r.Status)
		daysC := getDaysColor(r.DaysRemaining)

		fmt.Printf(" %d. %s%s%s\n", num, c, r.URL, ColReset)

		statusLine := fmt.Sprintf(" Status: %s%-5s%s", c, r.Status, ColReset)
		if !r.IsChainComplete {
			statusLine += fmt.Sprintf(" |%s CHAIN ISSUE%s", ColRed, ColReset)
		} else {
			statusLine += fmt.Sprintf(" |%s CHAIN VALID%s", ColGreen, ColReset)
		}
		fmt.Println(statusLine)

		fmt.Printf("   Days:%s%6d%s | Valid: %s → %s\n", daysC, r.DaysRemaining, ColReset,
			r.NotBefore.Format("02. Jan 2006"), r.NotAfter.Format("02. Jan 2006"))

		// chain details
		fmt.Printf("   Chain Length: %d Certificates\n", r.ChainLength)
		if r.IsSelfSigned {
			fmt.Printf("   %sSelf-Signed certificate%s\n", ColYellow, ColReset)
		}

		if !r.IsChainComplete && r.ChainError != "" {
			fmt.Printf("   %sError: %s%s\n", ColRed, r.ChainError, ColReset)
		} else if r.Error != nil {
			fmt.Printf("   %sError: %s%s\n", ColRed, r.ChainError, ColReset)
		}

		if r.RootIssuer != "" {
			fmt.Printf("   Root Issuer: %s\n", r.RootIssuer)
		}

		fmt.Printf("   Issuer: %s\n", r.Issuer)
		fmt.Printf("   Serial Number: %s\n", r.SerialNumber)

		// key info
		switch r.KeyAlgorithm {
		case "RSA":
			fmt.Printf("   Key: %s %d-bit | Sig: %s\n",
				r.KeyAlgorithm, r.KeySize, r.SignatureAlgorithm)
			if r.KeySize < 2048 {
				fmt.Printf("   %sWarning: Weak key size (%d bits)%s\n", ColYellow, r.KeySize, ColReset)
			}
			if r.KeySize >= 4096 {
				fmt.Printf("   %sInfo: Strong key size (%d bits)%s\n", ColGreen, r.KeySize, ColReset)
			}
			if r.KeySize == 2048 {
				fmt.Printf("   %sInfo: Acceptable key size (%d bits)%s\n", ColGreen, r.KeySize, ColReset)
			}
		case "ECDSA":
			if r.KeySize < 256 {
				fmt.Printf("   %sWarning: Weak key size (%d bits)%s\n", ColYellow, r.KeySize, ColReset)
			}
		}

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
		fmt.Printf("%s----------------------------------------%s\n", ColBlue, ColReset)
	}
	printSummary(results)
}

// ExportMarkdown saves the results as a Markdown table
func ExportMarkdown(results []checker.CertInfo, filename string) error {
	if filename == "" {
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// table title
	fmt.Fprintf(writer, "# Certificate Report\n\n")
	fmt.Fprintf(writer, "*Generated on: %s*\n\n", time.Now().Format("02. Jan 2006 15:04"))

	// table header
	fmt.Fprintln(writer, "| Domain | Status | Days Left | Issuer | Valid Until | Error |")
	fmt.Fprintln(writer, "| :--- | :--- | :---: | :--- | :--- | :--- |")

	// table rows
	for _, r := range results {
		errorMsg := ""
		if r.Error != nil {
			errorMsg = r.Error.Error()
		}
		// markdown escaping for pipe characters in data
		domain := strings.ReplaceAll(r.URL, "|", "\\|")
		issuer := strings.ReplaceAll(r.Issuer, "|", "\\|")
		errorMsg = strings.ReplaceAll(errorMsg, "|", "\\|")

		fmt.Fprintf(writer, "| %s | **%s** | %d | %s | %s | %s |\n",
			domain,
			r.Status,
			r.DaysRemaining,
			issuer,
			r.NotAfter.Format("02. Jan 2006"),
			errorMsg,
		)
	}

	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	total := len(results)

	if total == 0 {
		fmt.Fprint(writer, "\n*No results*\n")
		return nil
	}

	fmt.Fprintf(writer, "\n---\n\n### Summary\n")
	fmt.Fprintf(writer, "- **Total:** %d\n", total)
	fmt.Fprintf(writer, "- **Valid:** %d (%.1f%%)\n", counts["VALID"], float64(counts["VALID"])/float64(total)*100)
	fmt.Fprintf(writer, "- **Warnings:** %d (%.1f%%)\n", counts["WARNING"]+counts["SOON"], float64(counts["WARNING"]+counts["SOON"])/float64(total)*100)
	fmt.Fprintf(writer, "- **Expired:** %d (%.1f%%)\n", counts["EXPIRED"], float64(counts["EXPIRED"])/float64(total)*100)
	fmt.Fprintf(writer, "- **Errors:** %d (%.1f%%)\n", counts["ERROR"], float64(counts["ERROR"])/float64(total)*100)

	return nil
}

// TruncateString shortens a string and adds "..."
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
