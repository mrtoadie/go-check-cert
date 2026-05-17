// internal/checker/helper.go
package checker

import (
	"os"
	"strings"
)

// ExtractHostname removes protocols (http/https), paths, and ports
// to return a clean hostname.
// examples:
// "https://example.com/path" -> "example.com"
// "example.com:8443/api"     -> "example.com"
// "http://sub.domain.org"    -> "sub.domain.org"
func ExtractHostname(input string) string {
	if input == "" {
		return ""
	}
	// remove protocol
	host := strings.TrimPrefix(input, "https://")
	host = strings.TrimPrefix(host, "http://")
	// remove path (everything after the first '/')
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	// remove port (everything after the first ':')
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// determines known extensions (.pem, .crt, .cer, .key)
func IsCertFile(path string) bool {
	return strings.HasSuffix(path, ".pem") ||
		strings.HasSuffix(path, ".crt") ||
		strings.HasSuffix(path, ".cer") ||
		strings.HasSuffix(path, ".key")
}

// IsFilePath determines existence of the file on the local filesystem
func IsFilePath(path string) bool {
	if IsCertFile(path) {
		return true
	}
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
