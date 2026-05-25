// internal/config/config.go
package config

import (
	"bufio"
	"bytes"
	"cert-checker/internal/constants"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	cfg struct {
		Timeout     int
		UrlsFile    string
		LogFile     string
		OutputDir   string
		CertDir     string
		DefaultURLs []string
		WebPort     string
	}
	once sync.Once
)

// resolvePath resolves a path
// if inputPath is empty, defaultName is used in baseDir
func resolvePath(inputPath, baseDir, defaultName string) string {
	if inputPath == "" {
		return filepath.Join(baseDir, defaultName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(baseDir, defaultName) // fallback
	}

	if strings.HasPrefix(inputPath, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(inputPath, "~/"))
	}

	if filepath.IsAbs(inputPath) {
		return inputPath
	}

	return filepath.Join(baseDir, inputPath)
}

func loadConfig() {
	once.Do(func() {
		cfg.Timeout = 60 // default; overwritten if config.ini is present

		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("%sWarning: could not find home directory: %v%s\n", output.ColYellow, err, output.ColReset)
			return
		}

		configDir := filepath.Join(homeDir, constants.ConfigDir)

		// os.ReadFile returns IsNotExist — treat that as "no config yet", not an error
		data, err := os.ReadFile(filepath.Join(configDir, "config.ini"))
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("%sError reading config.ini: %v%s\n", output.ColRed, err, output.ColReset)
			return
		}

		if len(data) > 0 {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") ||
					(strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")) {
					continue
				}

				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}

				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])

				switch key {
				case "timeout":
					// validate: must be a positive integer
					if t, err := strconv.Atoi(val); err == nil && t > 0 {
						cfg.Timeout = t
					}
				case "urls_file":
					cfg.UrlsFile = val
				case "log_file":
					cfg.LogFile = val
				case "report_dir":
					cfg.OutputDir = val
				case "cert_dir":
					cfg.CertDir = val
				case "default_urls":
					cfg.DefaultURLs = parseURLs(val)
				case "web_port":
					if p, err := strconv.Atoi(val); err == nil && p >= 1 && p <= 65535 {
						cfg.WebPort = val
					}
				}
			}
		}

		// fallback: read default_urls.txt if default_urls was not set in the INI.
		if len(cfg.DefaultURLs) == 0 {
			urls, err := parser.ReadURLsFromFile(filepath.Join(configDir, "default_urls.txt"))
			if err != nil && !os.IsNotExist(err) {
				fmt.Printf("%sWarning: could not read default_urls.txt: %v%s\n", output.ColYellow, err, output.ColReset)
			}
			cfg.DefaultURLs = urls
		}

		// last-resort hardcoded fallback
		if len(cfg.DefaultURLs) == 0 {
			cfg.DefaultURLs = []string{"archlinux.org", "github.com", "go.dev"}
		}
	})
}

/*
func loadDefaultURLsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}
*/
// public getter
// GetTimeout returns the configured connection timeout in seconds.
func GetTimeout() int { loadConfig(); return cfg.Timeout }

// GetDefaultURLs returns the list of default URLs from configuration.
func GetDefaultURLs() []string { loadConfig(); return cfg.DefaultURLs }

func GetUrlsFile() string  { loadConfig(); return cfg.UrlsFile }
func GetLogFile() string   { loadConfig(); return cfg.LogFile }
func GetOutputDir() string { loadConfig(); return cfg.OutputDir }

// GetWebPort returns the configured web dashboard port.
// Falls back to defaultWebPort ("8080") if not set.
func GetWebPort() string {
	loadConfig()
	if cfg.WebPort == "" {
		return constants.DefaultWebPort
	}
	return cfg.WebPort
}

// configDir is a small helper to avoid repeating the homeDir+ConfigDir join
// in every path getter.
func configDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(homeDir, constants.ConfigDir), nil
}

// GetConfigPath returns the absolute path to the URL list file
func GetConfigPath() (string, error) {
	loadConfig()
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return resolvePath(cfg.UrlsFile, dir, "urls.txt"), nil
}

// GetLogPath returns the absolute path to the cron job log file
func GetLogPath() (string, error) {
	loadConfig()
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return resolvePath(cfg.LogFile, dir, "cert-check.log"), nil
}

// GetOutputPath returns the absolute path to the JSON report directory
func GetOutputPath() (string, error) {
	loadConfig()
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	/*
		outputDirRaw := cfg.OutputDir
		if outputDirRaw == "" {
			outputDirRaw = "reports"
		}
		if strings.HasPrefix(outputDirRaw, "~/") {
			return filepath.Join(homeDir, strings.TrimPrefix(outputDirRaw, "~/")), nil
		}
		if filepath.IsAbs(outputDirRaw) {
			return outputDirRaw, nil
			}*/
	//return filepath.Join(homeDir, constants.ConfigDir, outputDirRaw), nil
	//
	return resolvePath(cfg.OutputDir, dir, "reports"), nil
}

// GetCertPath returns the absolute path to the certificate download directory
func GetCertPath() (string, error) {
	loadConfig()
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	/*
		certDirRaw := cfg.CertDir
		if certDirRaw == "" {
			certDirRaw = "certs"
		}
		if strings.HasPrefix(certDirRaw, "~/") {
			return filepath.Join(homeDir, strings.TrimPrefix(certDirRaw, "~/")), nil
		}
		if filepath.IsAbs(certDirRaw) {
			return certDirRaw, nil
		}
		return filepath.Join(homeDir, constants.ConfigDir, certDirRaw), nil
	*/
	return resolvePath(cfg.CertDir, dir, "certs"), nil
}

// InitConfig loads the URL list from the configured path
func InitConfig() ([]string, error) {
	loadConfig()
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	finalURLPath := resolvePath(cfg.UrlsFile, dir, "urls.txt")

	if err := os.MkdirAll(filepath.Dir(finalURLPath), 0755); err != nil {
		return nil, fmt.Errorf("could not create config directory: %w", err)
	}

	_, statErr := os.Stat(finalURLPath)
	if statErr != nil && !os.IsNotExist(statErr) {
		// permission denied or similar — not a "file missing" situation
		return nil, fmt.Errorf("could not stat URL file %s: %w", finalURLPath, statErr)
	}

	if statErr == nil {
		// file exists — read it
		urls, err := parser.ReadURLsFromFile(finalURLPath)
		if err != nil {
			return nil, fmt.Errorf("error reading URL file %s: %w", finalURLPath, err)
		}
		return urls, nil
	}

	// first run: create the file
	fmt.Printf("%sFirst run: creating URL list at %s%s\n", output.ColBlue, finalURLPath, output.ColReset)
	file, err := os.Create(finalURLPath)
	if err != nil {
		return nil, fmt.Errorf("could not create URL file: %w", err)
	}
	defer file.Close()

	// bufio.Writer + explicit Flush — replaces the silent WriteString loop
	defaultURLs := GetDefaultURLs()
	w := bufio.NewWriter(file)
	for _, u := range defaultURLs {
		if _, err := fmt.Fprintf(w, "%s\n", u); err != nil {
			return nil, fmt.Errorf("could not write URL file: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return nil, fmt.Errorf("could not flush URL file: %w", err)
	}

	fmt.Printf("%sCreated:%s %s\n", output.ColGreen, output.ColReset, finalURLPath)
	fmt.Printf("%sTip:%s Edit this file to add your own URLs.\n\n", output.ColYellow, output.ColReset)

	return defaultURLs, nil
}

func parseURLs(s string) []string {
	var urls []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			urls = append(urls, p)
		}
	}
	return urls
}
