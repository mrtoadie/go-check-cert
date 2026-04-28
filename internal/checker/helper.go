// internal/checker/helper.go
// last modification: Apr 26 2026
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

// determines if the given string represents a local certificate file
// 1. known extensions (.pem, .crt, .cer, .key)
// 2. existence of the file on the local filesystem
func IsFilePath(input string) bool {
	if input == "" {
		return false
	}

	// 1. Check known extensions
	if strings.HasSuffix(input, ".pem") ||
		strings.HasSuffix(input, ".crt") ||
		strings.HasSuffix(input, ".cer") ||
		strings.HasSuffix(input, ".key") {
		return true
	}

	// 2. check existence (fallback for files without extension or wrong extension)
	_, err := os.Stat(input)
	return err == nil
}
