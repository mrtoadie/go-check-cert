// internal/parser/parser.go
package parser

import (
	"bufio"
	"os"
	"strings"

	"cert-checker/internal/checker"
)

// ParseInput decides centrally whether the input is a file or a list of URLs
func ParseInput(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		// Default behavior: Try to read urls.txt
		input = "urls.txt"
	}

	// central decision: Is it a file?
	if checker.IsFilePath(input) {
		// if it's a file, check if it's a certificate or a URL list
		if strings.HasSuffix(input, ".pem") ||
			strings.HasSuffix(input, ".crt") ||
			strings.HasSuffix(input, ".cer") ||
			strings.HasSuffix(input, ".key") {
			return []string{input}, nil // single certificate
		}
		// otherwise, it's a text file with URLs
		return ReadURLsFromFile(input)
	}

	// if not a file, treat as comma-separated URLs
	var urls []string
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			urls = append(urls, p)
		}
	}
	return urls, nil
}

// ReadURLsFromFile reads a list of URLs from a text file
func ReadURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
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