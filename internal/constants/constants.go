// internal/constants/constants.go
package constants

import "time"

const (
	AppName    = "cert-checker"
	Version    = "1.2.2"
	ConfigDir  = ".config/" + AppName
	CronMarker = "# cert-checker"
	// time format constants
	ReportDateFormat = "20060102-150405"     // report file names
	RFC3339Format    = time.RFC3339          // JSON timestamps
	CronDateFormat   = "2006-01-02 15:04:05" // cron job comments
)
