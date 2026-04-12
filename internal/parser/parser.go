package parser

import (
	"bufio"
	"os"
	"strings"
)

// decides whether a file is read or parsed as a list.
func ParseInput(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		input = "../config/urls.txt"
	}

	// check if it is an existing file
	if _, err := os.Stat(input); err == nil {
		return ReadURLsFromFile(input)
	}

	// otherwise treat as comma separated URLs
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

// reads URLs from a text file.
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
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}
