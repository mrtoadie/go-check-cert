// internal/checker/helper.go
package checker

import (
	"os"
	"strings"
)

// ExtractHostname returns the bare hostname from any supported URL format.
func ExtractHostname(input string) string {
	return parseTarget(input).Host
}

// IsCertFile determines known extensions .pem, .crt, .cer, .key
func IsCertFile(path string) bool {
	return strings.HasSuffix(path, ".pem") ||
		strings.HasSuffix(path, ".crt") ||
		strings.HasSuffix(path, ".cer") ||
		strings.HasSuffix(path, ".key")
}

// IsFilePath determines existence of the file on the local filesystem
func IsFilePath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CalculateExitCode calculates the exit code based on the certificate results
// rules:
// -EXPIRED or ERROR: Exit code 2 (critical error)
// -WARNING or SOON: Exit code 1 (warning)
// -Everything OK: exit code 0
func CalculateExitCode(results []CertInfo) int {
	hasWarning := false

	for _, r := range results {
		switch r.Status {
		case "EXPIRED", "ERROR":
			return 2 // return immediately as this is the highest priority
		case "WARNING", "SOON":
			hasWarning = true
		}
	}
	if hasWarning {
		return 1
	}
	return 0
}
