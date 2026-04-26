// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cert-checker/internal/output"
	"cert-checker/internal/parser"
)

const (
	ConfigDir  = ".config/cert-checker"
	ConfigFile = "urls.txt"
)

// checks whether the configuration file exists.
// returns the URLs and a bool (true = file recreated)
func InitConfig() ([]string, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, false, fmt.Errorf("could not find home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ConfigDir)
	configFile := filepath.Join(configPath, ConfigFile)

	// check if file exists
	if _, err := os.Stat(configFile); err == nil {
		// file exists > read it
		urls, err := parser.ReadURLsFromFile(configFile)
		return urls, false, err // false
	}

	// file does NOT exist > create it
	fmt.Print("First run: Creating configuration file...\n", output.ColBlue)

	// create directory
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, false, fmt.Errorf("could not create config directory: %w", err)
	}

	// default URLs
	defaultURLs := []string{
		"archlinux.org",
		"github.com",
		"ubuntu.com",
		"go.dev",
	}

	// write file
	file, err := os.Create(configFile)
	if err != nil {
		return nil, false, fmt.Errorf("could not create config file: %w", err)
	}
	defer file.Close()

	for _, url := range defaultURLs {
		if _, err := file.WriteString(url + "\n"); err != nil {
			return nil, false, fmt.Errorf("could not write URL: %w", err)
		}
	}

	fmt.Println("Created:", configFile, output.ColYellow)
	fmt.Printf("You can edit this file to add your own URLs.\n\n")

	return defaultURLs, true, nil
}
