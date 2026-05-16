// internal/config/defaults.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	//"strings"

	"cert-checker/internal/constants"
	"cert-checker/internal/output"
)

// EnsureDefaults erstellt config.ini und default_urls.txt, falls sie nicht existieren.
// Dies sollte am Anfang von main() aufgerufen werden.
func EnsureDefaults() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sWarning: Could not find home dir: %v%s\n", output.ColYellow, err, output.ColReset)
		return
	}

	configDir := filepath.Join(homeDir, constants.ConfigDir)
	
	// Sicherstellen, dass das Verzeichnis existiert
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("%sError creating config dir: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	// 1. config.ini erstellen, falls nicht vorhanden
	configIniPath := filepath.Join(configDir, "config.ini")
	if _, err := os.Stat(configIniPath); os.IsNotExist(err) {
		createConfigIni(configIniPath)
	}

	// 2. default_urls.txt erstellen, falls nicht vorhanden
	defaultURLsPath := filepath.Join(configDir, constants.DefaultURLsFile)
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
# If empty, ~/.config/cert-checker/urls.txt is used
urls_file = 

# Path to the log file
# If empty, ~/.config/cert-checker/cert-check.log is used
log_file = 

# Directory for JSON reports
# If empty, ~/.config/cert-checker/reports is used
output_dir = 

[settings]
# Timeout in seconds for network checks (default: 60)
timeout = 60

# Default first launch URLs (comma separated)
# If empty, default_urls.txt is used
default_urls = 

# Note: 
# -Empty values use the defaults from default_urls.txt or the code defaults.
# -Comments start with # or ;
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
ubuntu.com
go.dev
proton.me
cloudflare.com
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("%sError creating default_urls.txt: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	fmt.Printf("%sCreated:%s %s (Default URL source)%s\n", output.ColGreen, output.ColReset, path, output.ColReset)
}