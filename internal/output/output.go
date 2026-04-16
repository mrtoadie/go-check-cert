// internal/output/output.go
package output

import (
	"fmt"
	"cert-checker/internal/checker"
)

const (
	ColReset  = "\033[0m"
	ColRed    = "\033[31m"
	ColGreen  = "\033[32m"
	ColYellow = "\033[33m"
	ColBlue   = "\x1b[34m"
)

// selects the color based on the status of the certificate
func GetColor(status string) string {
	switch status {
	case "OK":
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

		fmt.Printf("%d. %s%s%s\n", num, c, r.URL, ColReset)
		fmt.Printf("   Days:%s%3d%s | Valid: %s → %s\n", daysC, r.DaysRemaining, ColReset,
			r.NotBefore.Format("02. Jan 2006"), r.NotAfter.Format("02. Jan 2006"))
		fmt.Printf("   Issuer: %s\n", r.Issuer)
		// new
		fmt.Printf("   Serialnumber:  %s\n", r.SerialNumber)
		//fmt.Printf("   Subject: %s\n", r.Subject)
		//
		// key info
		fmt.Printf("   Key:     %s %d-bit | Sig: %s\n", 
			r.KeyAlgorithm, r.KeySize, r.SignatureAlgorithm)
		//
		if r.Error != nil {
			fmt.Printf("   Error: %s%s%s\n", ColRed, r.Error, ColReset)
		}
		fmt.Printf("%s------------------------------------%s\n", ColBlue, ColReset)
	}

	// count summary
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}

	fmt.Printf("%s=== SUMMARY ===%s\n", ColBlue, ColReset)
	fmt.Printf("%sOK: %d%s | %sWarn: %d%s | %sExp: %d%s | %sErr: %d%s\n",
		ColGreen, counts["OK"], ColReset,
		ColYellow, counts["SOON"]+counts["WARNING"], ColReset,
		ColRed, counts["EXPIRED"], ColReset,
		ColRed, counts["ERROR"], ColReset)
}
