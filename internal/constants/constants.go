package constants

import "time"

const (
		AppName     = "cert-checker"
	ConfigDir   = ".config/" + AppName
	ConfigFile  = "urls.txt"
	LogFileName = "cert-check.log"
	CronMarker  = "# cert-checker"
	// time format constants
	ReportDateFormat = "20060102-150405"     // report file names
	RFC3339Format    = time.RFC3339          // JSON timestamps
	CronDateFormat   = "2006-01-02 15:04:05" // cron job comments
)
