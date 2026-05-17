// internal/web/dashboard.go
package web

import (
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cert-checker/internal/checker"
	"cert-checker/internal/config"
	"cert-checker/internal/constants"
)

//go:embed templates/dashboard.html
var templateFiles embed.FS

type DashboardPage struct {
	LastUpdated  time.Time
	Total        int
	OK           int
	Warn         int
	Exp          int
	Err          int
	Results      []checker.CertInfo
	ErrorMessage string
	Version      string
}

// StartServer started the web server (HTTP oder HTTPS)
func StartServer(port, certFile, keyFile, Version string) {

	initTemplate()

	// route for the main page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderDashboard(w, r)
	})

	// route for JSON API
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		isSecure := certFile != "" && keyFile != ""
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"status": "running", "secure": %t, "port": "%s"}`, isSecure, port)))
	})

	addr := ":" + port
	// HTTPS with local self signed cert
	if certFile != "" && keyFile != "" {
		// check whether files exist
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			log.Fatalf("Certificate file not found: %s", certFile)
		}
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			log.Fatalf("Key file not found: %s", keyFile)
		}

		log.Printf("Secure Dashboard starting on https://localhost:%s", port)
		log.Printf("Certificate: %s", certFile)
		log.Printf("Key: %s", keyFile)

		// TLS
		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12, // enforces at least TLS 1.2
			PreferServerCipherSuites: true,
		}

		server := &http.Server{
			Addr:      addr,
			TLSConfig: tlsConfig,
		}

		// https mode
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Fatalf("HTTPS Server failed: %v", err)
		}

	} else {
		// http mode
		log.Printf("Insecure Dashboard starting on http://localhost:%s", port)
		log.Printf("Tip: Use --cert and --key flags for HTTPS protection!")

		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("HTTP Server failed: %v", err)
		}
	}
}

// global template var
var dashboardTemplate *template.Template

func initTemplate() {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"add":   func(a, b float64) float64 { return a + b },
	}

	// html dashboard template
	content, err := templateFiles.ReadFile("templates/dashboard.html")
	if err != nil {
		log.Fatalf("Failed to read template file: %v", err)
	}

	var errParse error
	dashboardTemplate, errParse = template.New("dashboard").Funcs(funcMap).Parse(string(content))
	if errParse != nil {
		log.Fatalf("Failed to parse template: %v", errParse)
	}
}

func renderDashboard(w http.ResponseWriter, r *http.Request) {
	var page DashboardPage
	page.LastUpdated = time.Now()
	page.Version = constants.Version

	// get output directory from config
	outputDir, err := config.GetOutputPath()
	if err != nil {
		page.ErrorMessage = fmt.Sprintf("Config error: %v", err)
		renderPage(w, page)
		return
	}

	log.Printf("Scanning directory: %s", outputDir)

	// check whether the directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {		
		page.ErrorMessage = fmt.Sprintf("Output directory not found: %s. Run 'cert-checker' first to generate reports.", outputDir)
		renderPage(w, page)
		return
	}

	// search reports
	files, err := filepath.Glob(filepath.Join(outputDir, "cert-report-*.json"))
	if err != nil {
		page.ErrorMessage = fmt.Sprintf("Error scanning directory: %v", err)
		renderPage(w, page)
		return
	}

	if len(files) == 0 {
		page.ErrorMessage = "No reports found. Run 'cert-checker' first to generate reports."
		renderPage(w, page)
		return
	}

	// sort (newest first)
	sort.Strings(files)
	log.Printf("Found %d report files. Merging...", len(files))

	latestResults := make(map[string]checker.CertInfo)
	var allGeneratedTimes []time.Time

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Skipping unreadable file %s: %v", file, err)
			continue
		}

		var report struct {
			GeneratedAt string `json:"generated_at"`
			TotalCount  int    `json:"total_count"`
			Results     []struct {
				URL                string      `json:"URL"`
				Issuer             string      `json:"Issuer"`
				Subject            string      `json:"Subject"`
				SerialNumber       string      `json:"SerialNumber"`
				NotBefore          string      `json:"NotBefore"`
				NotAfter           string      `json:"NotAfter"`
				DaysRemaining      int         `json:"DaysRemaining"`
				Status             string      `json:"Status"`
				Error              interface{} `json:"Error"`
				KeyAlgorithm       string      `json:"KeyAlgorithm"`
				KeySize            int         `json:"KeySize"`
				SignatureAlgorithm string      `json:"SignatureAlgorithm"`
				SANs               []string    `json:"SANs"`
				ChainLength        int         `json:"ChainLength"`
				IsChainComplete    bool        `json:"IsChainComplete"`
				ChainError         string      `json:"ChainError"`
				IsSelfSigned       bool        `json:"IsSelfSigned"`
				RootIssuer         string      `json:"RootIssuer"`
			} `json:"results"`
		}

		if err := json.Unmarshal(data, &report); err != nil {
			log.Printf("Skipping invalid JSON %s: %v", file, err)
			continue
		}

		if t, err := time.Parse(time.RFC3339, report.GeneratedAt); err == nil {
			allGeneratedTimes = append(allGeneratedTimes, t)
		}

		for _, res := range report.Results {
			notBefore, _ := time.Parse(time.RFC3339, res.NotBefore)
			notAfter, _ := time.Parse(time.RFC3339, res.NotAfter)
			
			// Error Handling (String oder nil)
			var err error
			switch v := res.Error.(type) {
			case string:
				if v != "" {
					err = fmt.Errorf(v)
				}
			case nil:
				err = nil
			default:
				// Andere Typen ignorieren oder als Fehler behandeln
				err = fmt.Errorf("unexpected error format")
			}

			certInfo := checker.CertInfo{
				URL:                res.URL,
				Issuer:             res.Issuer,
				Subject:            res.Subject,
				SerialNumber:       res.SerialNumber,
				NotBefore:          notBefore,
				NotAfter:           notAfter,
				DaysRemaining:      res.DaysRemaining,
				Status:             res.Status,
				Error:              err,
				KeyAlgorithm:       res.KeyAlgorithm,
				KeySize:            res.KeySize,
				SignatureAlgorithm: res.SignatureAlgorithm,
				SANs:               res.SANs,
				ChainLength:        res.ChainLength,
				IsChainComplete:    res.IsChainComplete,
				ChainError:         res.ChainError,
				IsSelfSigned:       res.IsSelfSigned,
				RootIssuer:         res.RootIssuer,
			}
			latestResults[res.URL] = certInfo
		}
	}

	var mergedResults []checker.CertInfo
	for _, res := range latestResults {
		mergedResults = append(mergedResults, res)
	}

	// Sortieren nach URL
	sort.Slice(mergedResults, func(i, j int) bool {
		return mergedResults[i].URL < mergedResults[j].URL
	})

	// Bestes LastUpdated finden
	if len(allGeneratedTimes) > 0 {
		sort.Slice(allGeneratedTimes, func(i, j int) bool {
			return allGeneratedTimes[i].After(allGeneratedTimes[j])
		})
		page.LastUpdated = allGeneratedTimes[0]
	}

	log.Printf("Merged %d unique domains from %d files.", len(mergedResults), len(files))

	page.Total = len(mergedResults)
	page.Results = mergedResults

	for _, res := range mergedResults {
		switch res.Status {
		case "VALID":
			page.OK++
		case "WARNING", "SOON":
			page.Warn++
		case "EXPIRED":
			page.Exp++
		case "ERROR":
			page.Err++
		}
	}

	renderPage(w, page)
}

func renderPage(w http.ResponseWriter, page DashboardPage) {
	if err := dashboardTemplate.ExecuteTemplate(w, "dashboard", page); err != nil {
		http.Error(w, "Render error: "+err.Error(), http.StatusInternalServerError)
	}
}
