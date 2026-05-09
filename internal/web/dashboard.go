// internal/web/dashboard.go
package web

import (
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
)

// structure for the template
type DashboardPage struct {
	LastUpdated  time.Time
	Total        int
	OK           int
	Warn         int
	Exp          int
	Err          int
	Results      []checker.CertInfo
	ErrorMessage string // error message in UI
}

// StartServer
func StartServer(port string) {
	// route for the main page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// optional: if ?refresh=true, you could start a new check here
		// currently it only reloads the latest file
		renderDashboard(w, r)
	})

	// route for JSON API
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "running", "message": "Use the dashboard to view details"}`))
	})

	log.Printf("🚀 Dashboard started on http://localhost:%s", port)
	log.Printf("💡 Tip: Run 'cert-checker -ci' first to generate a report!")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func renderDashboard(w http.ResponseWriter, r *http.Request) {
	var page DashboardPage
	page.LastUpdated = time.Now()

	// find JSON file
	configPath, err := config.GetConfigPath()
	if err != nil {
		page.ErrorMessage = fmt.Sprintf("Config error: %v", err)
		renderPage(w, page)
		return
	}

	reportDir := filepath.Dir(configPath)
	log.Printf("🔍 Searching for reports in: %s", reportDir)

	// find all JSON files
	files, err := filepath.Glob(filepath.Join(reportDir, "cert-report-*.json"))
	if err != nil {
		page.ErrorMessage = fmt.Sprintf("Error scanning directory: %v", err)
		renderPage(w, page)
		return
	}

	if len(files) == 0 {
		page.ErrorMessage = "No reports found. Please run 'cert-checker -ci' or save a report first."
		renderPage(w, page)
		return
	}

	// sort descending (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	lastFile := files[0]

	log.Printf("📂 Using latest report: %s", lastFile)

	// 2. JSON lesen
	data, err := os.ReadFile(lastFile)
	if err != nil {
		page.ErrorMessage = fmt.Sprintf("Could not read report: %v", err)
		renderPage(w, page)
		return
	}

	if len(data) == 0 {
		page.ErrorMessage = "Report file is empty."
		renderPage(w, page)
		return
	}

	// JSON parsen
	var report struct {
		GeneratedAt string             `json:"generated_at"`
		TotalCount  int                `json:"total_count"`
		Results     []checker.CertInfo `json:"results"`
	}

	if err := json.Unmarshal(data, &report); err != nil {
		log.Printf("❌ JSON Parse Error: %v", err)
		log.Printf("📄 First 200 chars: %s", string(data[:min(200, len(data))]))
		page.ErrorMessage = fmt.Sprintf("Invalid JSON format: %v", err)
		renderPage(w, page)
		return
	}

	log.Printf("✅ JSON Parsed successfully. Count: %d", report.TotalCount)

	// calculate statistics
	page.Total = report.TotalCount
	page.Results = report.Results

	for _, r := range report.Results {
		switch r.Status {
		case "OK":
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

// renderPage renders the HTML template
func renderPage(w http.ResponseWriter, page DashboardPage) {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	tmpl, err := template.New("dashboard").Funcs(funcMap).Parse(dashboardHTML)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, page); err != nil {
		http.Error(w, "Render error: "+err.Error(), http.StatusInternalServerError)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// dashboardHTML template
const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cert-Checker Dashboard</title>
    <style>
        /* CSS Variablen für Light Mode (Default) */
        :root {
            --bg-color: #f4f6f8;
            --container-bg: #ffffff;
            --text-color: #333333;
            --heading-color: #2c3e50;
            --border-color: #eeeeee;
            --table-header-bg: #f8f9fa;
            --table-hover-bg: #f8f9fa;
            --footer-color: #6c757d;
            --shadow: 0 4px 6px rgba(0,0,0,0.05);
            --error-bg: #fff3cd;
            --error-text: #856404;
            --error-border: #ffc107;
        }

        /* Dark Mode Overrides */
        body.dark-mode {
            --bg-color: #1a1a2e;
            --container-bg: #16213e;
            --text-color: #e0e0e0;
            --heading-color: #ffffff;
            --border-color: #2a2a4a;
            --table-header-bg: #0f3460;
            --table-hover-bg: #1f4068;
            --footer-color: #a0a0a0;
            --shadow: 0 4px 6px rgba(0,0,0,0.3);
            --error-bg: #3d2b1f;
            --error-text: #ffc107;
            --error-border: #ffc107;
        }

        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; 
            background: var(--bg-color); 
            padding: 20px; 
            color: var(--text-color);
            transition: background 0.3s ease, color 0.3s ease;
        }

        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            background: var(--container-bg); 
            padding: 30px; 
            border-radius: 12px; 
            box-shadow: var(--shadow);
            transition: background 0.3s ease, box-shadow 0.3s ease;
        }

        h1 { 
            color: var(--heading-color); 
            margin-bottom: 30px; 
            font-size: 2rem; 
            display: flex; 
            align-items: center; 
            justify-content: space-between;
            gap: 10px; 
        }

        /* Dark Mode Toggle Button */
        .theme-toggle {
            background: transparent;
            border: 2px solid var(--text-color);
            color: var(--text-color);
            padding: 8px 16px;
            border-radius: 20px;
            cursor: pointer;
            font-size: 0.9rem;
            font-weight: 600;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .theme-toggle:hover {
            background: var(--text-color);
            color: var(--bg-color);
        }

        .stats { display: flex; gap: 20px; margin-bottom: 30px; flex-wrap: wrap; }
        .card { flex: 1; min-width: 140px; padding: 20px; border-radius: 8px; text-align: center; color: white; font-weight: bold; font-size: 1.2rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .bg-ok { background: linear-gradient(135deg, #28a745, #218838); }
        .bg-warn { background: linear-gradient(135deg, #ffc107, #e0a800); color: #333; }
        .bg-exp { background: linear-gradient(135deg, #dc3545, #c82333); }
        .bg-err { background: linear-gradient(135deg, #6c757d, #5a6268); }
        
        .error-box { background: var(--error-bg); color: var(--error-text); padding: 15px; border-radius: 6px; border-left: 5px solid var(--error-border); margin-bottom: 20px; }
        
        table { width: 100%; border-collapse: collapse; margin-top: 20px; font-size: 0.95rem; }
        th, td { padding: 14px; text-align: left; border-bottom: 1px solid var(--border-color); }
        th { background: var(--table-header-bg); color: var(--text-color); font-weight: 600; text-transform: uppercase; font-size: 0.85rem; letter-spacing: 0.5px; }
        tr:hover { background-color: var(--table-hover-bg); }
        
        .status-ok { color: #28a745; font-weight: 700; }
        .status-warning, .status-soon { color: #d39e00; font-weight: 700; }
        .status-expired { color: #dc3545; font-weight: 700; }
        .status-error { color: #6c757d; font-weight: 700; }
        
        .btn-refresh { background: #007bff; color: white; border: none; padding: 12px 24px; border-radius: 6px; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; gap: 8px; font-weight: 500; transition: background 0.2s; }
        .btn-refresh:hover { background: #0056b3; }
        
        .footer { margin-top: 30px; color: var(--footer-color); font-size: 0.9rem; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <h1>
            🔒 Certificate Dashboard
            <button id="themeToggle" class="theme-toggle" onclick="toggleTheme()">
                <span id="themeIcon">🌙</span> <span id="themeText">Dark Mode</span>
            </button>
        </h1>
        
        {{if .ErrorMessage}}
        <div class="error-box">
            <strong>⚠️ {{.ErrorMessage}}</strong>
        </div>
        {{else}}
        
        <div class="stats">
            <div class="card bg-ok">OK: {{.OK}}</div>
            <div class="card bg-warn">Warn: {{.Warn}}</div>
            <div class="card bg-exp">Expired: {{.Exp}}</div>
            <div class="card bg-err">Errors: {{.Err}}</div>
        </div>

        <a href="/" class="btn-refresh">🔄 Refresh Data</a>

        <table>
            <thead>
                <tr>
                    <th>Domain / File</th>
                    <th>Status</th>
                    <th>Days Left</th>
                    <th>Issuer</th>
                    <th>Valid Until</th>
                </tr>
            </thead>
            <tbody>
                {{if eq (len .Results) 0}}
                <tr>
                    <td colspan="5" style="text-align: center; color: var(--footer-color); padding: 40px;">No certificate data available.</td>
                </tr>
                {{else}}
                {{range .Results}}
                <tr>
                    <td><strong>{{.URL}}</strong></td>
                    <td class="status-{{lower .Status}}">{{.Status}}</td>
                    <td>{{.DaysRemaining}}</td>
                    <td>{{.Issuer}}</td>
                    <td>{{.NotAfter.Format "02. Jan 2006"}}</td>
                </tr>
                {{end}}
                {{end}}
            </tbody>
        </table>
        {{end}}
        
        <div class="footer">
            Last updated: {{.LastUpdated.Format "02. Jan 2006 15:04:05"}}
        </div>
    </div>

    <script>
        // Theme Toggle Logic
        const toggleBtn = document.getElementById('themeToggle');
        const themeIcon = document.getElementById('themeIcon');
        const themeText = document.getElementById('themeText');
        const body = document.body;

        // Check localStorage on load
        const savedTheme = localStorage.getItem('cert-dashboard-theme');
        if (savedTheme === 'dark') {
            body.classList.add('dark-mode');
            updateButton(true);
        }

        function toggleTheme() {
            body.classList.toggle('dark-mode');
            const isDark = body.classList.contains('dark-mode');
            localStorage.setItem('cert-dashboard-theme', isDark ? 'dark' : 'light');
            updateButton(isDark);
        }

        function updateButton(isDark) {
            if (isDark) {
                themeIcon.textContent = '☀️';
                themeText.textContent = 'Light Mode';
            } else {
                themeIcon.textContent = '🌙';
                themeText.textContent = 'Dark Mode';
            }
        }
    </script>
</body>
</html>
`
