// internal/constants/constants.go
package constants

import "time"

const (
	AppName        = "cert-checker"
	Version        = "1.4.0"
	ConfigDir      = ".config/" + AppName
	CronMarker     = "# cert-checker"
	DefaultWebPort = "8080"
	// time format constants
	ReportDateFormat = "20060102-150405"     // report file names
	RFC3339Format    = time.RFC3339          // JSON timestamps
	CronDateFormat   = "2006-01-02 15:04:05" // cron job comments
	// color constants
	CriticalThresholdDays = 30
	WarningThresholdDays  = 60
	ColReset              = "\033[0m"
	ColRed                = "\033[31m"
	ColGreen              = "\033[32m"
	ColYellow             = "\033[33m"
	ColBlue               = "\033[34m"
)
