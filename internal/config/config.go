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
		DefaultURLs []string
		Loaded      bool
	}
	once sync.Once
)

// resolvePath löst einen Pfad auf (handle ~, absolute, relative Pfade)
func resolvePath(inputPath, baseDir, defaultName string) string {
	if inputPath == "" {
		return filepath.Join(baseDir, defaultName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(baseDir, defaultName)
	}

	if strings.HasPrefix(inputPath, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(inputPath, "~/"))
	}

	if filepath.IsAbs(inputPath) {
		return inputPath
	}

	// Relativer Pfad oder nur Dateiname -> in baseDir legen
	return filepath.Join(baseDir, inputPath)
}

func loadConfig() {
	once.Do(func() {
		// 1. DEFAULTS SETZEN
		cfg.Timeout = constants.DefaultTimeout
		cfg.UrlsFile = ""
		cfg.LogFile = ""
		cfg.OutputDir = "reports"
		cfg.Loaded = true

		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("%sWarning: Could not find home dir: %v%s\n", output.ColYellow, err, output.ColReset)
			return
		}

		configDir := filepath.Join(homeDir, constants.ConfigDir)
		configIniPath := filepath.Join(configDir, "config.ini")

		// 2. INI LADEN
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
				case "output_dir":
					cfg.OutputDir = val
				case "default_urls":
					cfg.DefaultURLs = parseURLs(val)
				}
			}
		}

		// 3. FALLBACK: default_urls.txt laden (nur wenn INI nichts gesetzt hat)
		if len(cfg.DefaultURLs) == 0 {
			defaultURLsPath := filepath.Join(configDir, constants.DefaultURLsFile)
			if _, err := os.Stat(defaultURLsPath); err == nil {
				cfg.DefaultURLs = loadDefaultURLsFromFile(defaultURLsPath)
				if len(cfg.DefaultURLs) > 0 {
					fmt.Printf("%sInfo: Loaded %d default URLs from %s%s\n",
						output.ColBlue, len(cfg.DefaultURLs), constants.DefaultURLsFile, output.ColReset)
				}
			}
		}

		// 4. LETZTER FALLBACK: Hartkodierte URLs
		if len(cfg.DefaultURLs) == 0 {
			cfg.DefaultURLs = []string{"archlinux.org", "github.com", "ubuntu.com", "go.dev"}
		}
	})
}

// loadDefaultURLsFromFile liest URLs aus einer Datei
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

// --- PUBLIC GETTER (rufen loadConfig() auf) ---

func GetTimeout() int {
	loadConfig()
	return cfg.Timeout
}

func GetUrlsFile() string {
	loadConfig()
	return cfg.UrlsFile
}

func GetLogFile() string {
	loadConfig()
	return cfg.LogFile
}

func GetOutputDir() string {
	loadConfig()
	return cfg.OutputDir
}

func GetDefaultURLs() []string {
	loadConfig()
	return cfg.DefaultURLs
}

// --- INITCONFIG (MUSST loadConfig() AUFRUFEN!) ---

func InitConfig() ([]string, bool, error) {
	// WICHTIG: Config zuerst laden!
	loadConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, false, fmt.Errorf("could not find home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, constants.ConfigDir)

	// NUTZE GETTER statt direktem cfg.Zugriff!
	urlsFile := GetUrlsFile()
	finalUrlPath := resolvePath(urlsFile, configDir, constants.DefaultURLsFile)

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

	// Datei erstellen mit den geladenen Defaults
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

// --- WRAPPER FÜR ALTE AUFRUFE ---

func GetConfigPath() (string, error) {
	loadConfig()
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, constants.ConfigDir)
	return resolvePath(cfg.UrlsFile, configDir, constants.DefaultURLsFile), nil
}

func GetLogPath() (string, error) {
	loadConfig()
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, constants.ConfigDir)
	return resolvePath(cfg.LogFile, configDir, constants.LogFileName), nil
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