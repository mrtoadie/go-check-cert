// internal/config/config.go
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"cert-checker/internal/constants"
	"cert-checker/internal/output"
	"cert-checker/internal/parser"
)

var (
	cfg struct {
		Timeout     int
		UrlsFile    string
		LogFile     string
		OutputDir   string
		CertDir     string
		DefaultURLs []string
		Loaded      bool
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
		// set defaults (only as a fallback for the INI read logic, not for the path)
		cfg.Timeout = 60 // fallback
		cfg.Loaded = true

		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("%sWarning: Could not find home dir: %v%s\n", output.ColYellow, err, output.ColReset)
			return
		}

		configDir := filepath.Join(homeDir, constants.ConfigDir)
		configIniPath := filepath.Join(configDir, "config.ini")

		// load INI
		data, err := os.ReadFile(configIniPath)
		hasIni := true
		if err != nil {
			if os.IsNotExist(err) {
				hasIni = false
			} else {
				fmt.Printf("%sError reading config: %v%s\n", output.ColRed, err, output.ColReset)
				return
			}
		}

		if hasIni {
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
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
					if t, err := strconv.Atoi(val); err == nil {
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
					cfg.WebPort = val
				}
			}
		}

		// fallback: load default_urls.txt (only if INI has not set anything)
		if len(cfg.DefaultURLs) == 0 {
			defaultURLsPath := filepath.Join(configDir, "default_urls.txt")
			if _, err := os.Stat(defaultURLsPath); err == nil {
				cfg.DefaultURLs = loadDefaultURLsFromFile(defaultURLsPath)
			}
		}

		// fallback: hardcoded URLs (only if there is nothing there)
		if len(cfg.DefaultURLs) == 0 {
			cfg.DefaultURLs = []string{"archlinux.org", "github.com", "go.dev"}
		}
	})
}

func loadDefaultURLsFromFile(filePath string) []string {
	var urls []string
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls
}

// public getter
func GetTimeout() int          { loadConfig(); return cfg.Timeout }
func GetUrlsFile() string      { loadConfig(); return cfg.UrlsFile }
func GetLogFile() string       { loadConfig(); return cfg.LogFile }
func GetOutputDir() string     { loadConfig(); return cfg.OutputDir }
func GetDefaultURLs() []string { loadConfig(); return cfg.DefaultURLs }

func GetWebPort() string {
	loadConfig()
	if cfg.WebPort == "" {
		return "8080"
	}
	return cfg.WebPort
}

// InitConfig
func InitConfig() ([]string, bool, error) {
	loadConfig()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, false, fmt.Errorf("could not find home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, constants.ConfigDir)

	urlsFile := GetUrlsFile()
	finalUrlPath := resolvePath(urlsFile, configDir, "urls.txt")

	if err := os.MkdirAll(filepath.Dir(finalUrlPath), 0755); err != nil {
		return nil, false, fmt.Errorf("could not create config directory: %w", err)
	}

	if _, err := os.Stat(finalUrlPath); err == nil {
		urls, err := parser.ReadURLsFromFile(finalUrlPath)
		if err != nil {
			return nil, false, fmt.Errorf("error reading URL file %s: %w", finalUrlPath, err)
		}
		return urls, false, nil
	}

	fmt.Printf("%sFirst run: Creating configuration file at %s...%s\n", output.ColBlue, finalUrlPath, output.ColReset)
	file, err := os.Create(finalUrlPath)
	if err != nil {
		return nil, false, fmt.Errorf("could not create URL file: %w", err)
	}
	defer file.Close()

	defaultURLs := GetDefaultURLs()
	for _, url := range defaultURLs {
		file.WriteString(url + "\n")
	}

	fmt.Printf("%sCreated:%s %s\n", output.ColGreen, output.ColReset, finalUrlPath)
	fmt.Printf("%sTip:%s Edit this file to add your own URLs.\n\n", output.ColYellow, output.ColReset)

	return defaultURLs, true, nil
}

func parseURLs(s string) []string {
	var urls []string
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			urls = append(urls, p)
		}
	}
	return urls
}

// wrapper
func GetConfigPath() (string, error) {
	loadConfig()
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, constants.ConfigDir)
	return resolvePath(cfg.UrlsFile, configDir, "urls.txt"), nil
}

func GetLogPath() (string, error) {
	loadConfig()
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, constants.ConfigDir)
	return resolvePath(cfg.LogFile, configDir, "cert-check.log"), nil
}

func GetOutputPath() (string, error) {
	loadConfig()
	homeDir, _ := os.UserHomeDir()
	outputDirRaw := cfg.OutputDir
	if outputDirRaw == "" {
		outputDirRaw = "reports"
	}
	if strings.HasPrefix(outputDirRaw, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(outputDirRaw, "~/")), nil
	}
	if filepath.IsAbs(outputDirRaw) {
		return outputDirRaw, nil
	}
	return filepath.Join(homeDir, constants.ConfigDir, outputDirRaw), nil
}

func GetCertPath() (string, error) {
	loadConfig()
	homeDir, _ := os.UserHomeDir()
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
}
