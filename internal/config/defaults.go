// internal/config/defaults.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cert-checker/internal/constants"
	"cert-checker/internal/output"
)

// EnsureDefaults creates config.ini and default_urls.txt if they do not exist.
// called at the beginning of main().
func EnsureDefaults() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sWarning: Could not find home dir: %v%s\n", output.ColYellow, err, output.ColReset)
		return
	}

	configDir := filepath.Join(homeDir, constants.ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("%sError creating config dir: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	configIniPath := filepath.Join(configDir, "config.ini")
	if _, err := os.Stat(configIniPath); os.IsNotExist(err) {
		createConfigIni(configIniPath)
	}
	/*
		logsDir := filepath.Join(configDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			fmt.Printf("%sError creating logs dir: %v%s\n", output.ColRed, err, output.ColReset)
			return
		}

		reportsDir := filepath.Join(configDir, "reports")
		if err := os.MkdirAll(reportsDir, 0755); err != nil {
			fmt.Printf("%sError creating reports dir: %v%s\n", output.ColRed, err, output.ColReset)
			return
		}

		certsDir := filepath.Join(configDir, "certs")
		if err := os.MkdirAll(certsDir, 0755); err != nil {
			fmt.Printf("%sError creating certs dir: %v%s\n", output.ColRed, err, output.ColReset)
			return
		}
	*/
	for _, dir := range []string{"logs", "reports", "certs"} {
		if err := os.MkdirAll(filepath.Join(configDir, dir), 0755); err != nil {
			fmt.Printf("%sError creating %s dir: %v%s\n", output.ColRed, dir, err, output.ColReset)
		}
	}

	defaultURLsPath := filepath.Join(configDir, "default_urls.txt")
	if _, err := os.Stat(defaultURLsPath); os.IsNotExist(err) {
		createDefaultURLs(defaultURLsPath)
	}
}

func createConfigIni(path string) {
	content := `# cert-checker Configuration File
# Created automatically on first run
# Lines starting with '#' are comments

[paths]
# Path to URL list (supports ~, /or relative path)
urls_file = ~/.config/cert-checker/default_urls.txt

# Path to the log file
log_file = ~/.config/cert-checker/logs/cert-check.log

# Directory for JSON reports
report_dir = ~/.config/cert-checker/reports

# Directory for certificate files
cert_dir = ~/.config/cert-checker/certs

[settings]
# Timeout in seconds for network checks (default: 60)
timeout = 60
# Default webserver port
web_port = 8080
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("%sError creating config.ini: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	fmt.Printf("%sCreated:%s %s\n", output.ColGreen, output.ColReset, path)
	fmt.Printf("%sTip:%s Edit this file to customize settings.%s\n\n", output.ColYellow, output.ColReset, output.ColReset)
}

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
		fmt.Printf("%sError creating default_urls.txt: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}
	fmt.Printf("%sCreated:%s %s (Default URL source)%s\n", output.ColGreen, output.ColReset, path, output.ColReset)
}
