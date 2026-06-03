// internal/config/defaults.go
package config

import (
	"cert-checker/internal/constants"
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDefaults creates config.ini and default_urls.txt if they do not exist
func EnsureDefaults() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sWarning: could not find home dir: %v%s\n", constants.ColYellow, err, constants.ColReset)
		return
	}

	configDir := filepath.Join(homeDir, constants.ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("%sError creating config dir: %v%s\n", constants.ColRed, err, constants.ColReset)
		return
	}

	configIniPath := filepath.Join(configDir, "config.ini")
	if _, err := os.Stat(configIniPath); os.IsNotExist(err) {
		createConfigIni(configIniPath)
	}

	// create sub-directories independently — a failure on one does not
	// prevent the others from being created (previously used early returns)
	for _, dir := range []string{"logs", "reports", "certs"} {
		if err := os.MkdirAll(filepath.Join(configDir, dir), 0755); err != nil {
			fmt.Printf("%sError creating %s dir: %v%s\n", constants.ColRed, dir, err, constants.ColReset)
		}
	}

	if _, err := os.Stat(filepath.Join(configDir, "default_urls.txt")); os.IsNotExist(err) {
		createDefaultURLs(filepath.Join(configDir, "default_urls.txt"))
	}
}

// createConfigIni creates the config.ini file with default values
func createConfigIni(path string) {
	// base is built from the constant so the template stays in sync if ConfigDir changes
	base := "~/" + constants.ConfigDir
	content := fmt.Sprintf(`# cert-checker Configuration File
# Created automatically on first run
# Lines starting with '#' are comments

[paths]
# Path to URL list (supports ~/, absolute, or relative path)
urls_file = %s/default_urls.txt

# Path to the log file
log_file = %s/logs/cert-check.log

# Directory for JSON reports
report_dir = %s/reports

# Directory for certificate files
cert_dir = %s/certs

[settings]
# Timeout in seconds for network checks (default: 60)
timeout = 60
# Web dashboard port (default: %s)
web_port = %s
`, base, base, base, base, constants.DefaultWebPort, constants.DefaultWebPort)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("%sError creating config.ini: %v%s\n", constants.ColRed, err, constants.ColReset)
		return
	}
	fmt.Printf("%sCreated:%s %s\n", constants.ColGreen, constants.ColReset, path)
	fmt.Printf("%sTip:%s Edit this file to customize settings.\n\n", constants.ColYellow, constants.ColReset)
	// removed redundant trailing ColReset
}

// createDefaultURLs creates the default_urls.txt file with default URLs
func createDefaultURLs(path string) {
	content := `# Default URLs for cert-checker
# These are used if 'default_urls' is not set in config.ini.
# Add or remove domains here to change the defaults.

archlinux.org
github.com
go.dev
cloudflare.com
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("%sError creating default_urls.txt: %v%s\n", constants.ColRed, err, constants.ColReset)
		return
	}
	fmt.Printf("%sCreated:%s %s (Default URL source)\n", constants.ColGreen, constants.ColReset, path)
}
