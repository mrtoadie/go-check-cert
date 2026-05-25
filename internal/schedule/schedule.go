// internal/schedule/schedule.go
package schedule

import (
	"cert-checker/internal/config"
	"cert-checker/internal/constants"
	"cert-checker/internal/output"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

// CronJob represents a cron entry
type CronJob struct {
	Comment   string
	Command   string
	Index     int
	FullEntry string
}

// ScheduleMain starts the cron setup menu
func ScheduleMain() {
	var action string

	fmt.Printf("%s=== CRON JOB SETUP ===%s\n", output.ColBlue, output.ColReset)

	urlsPath, _ := config.GetConfigPath()
	logPath, _ := config.GetLogPath()

	fmt.Printf("%sNote: This job automatically checks the URLs from your configuration file.%s\n", output.ColYellow, output.ColReset)
	fmt.Printf("%sURLs File: %s%s\n", output.ColBlue, urlsPath, output.ColReset)
	fmt.Printf("%sLog File:  %s%s\n\n", output.ColBlue, logPath, output.ColReset)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose action").
				Description("What do you want to do?").
				Options(
					huh.NewOption("Set up cron job", "install"),
					huh.NewOption("List & remove cron jobs", "list"),
					huh.NewOption("Exit", "exit"),
				).
				Value(&action),
		).WithTheme(huh.ThemeBase16()),
	)

	if err := form.Run(); err != nil {
		fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
		return
	}

	switch action {
	case "install":
		CreateCron()
	case "list":
		ListAndManageJobs()
	case "exit":
		fmt.Println("Bye...")
	}
}

// validateCronField is a generic validation function
func validateCronField(value string, min, max int, allowWildcard bool) error {
	if value == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if allowWildcard && strings.HasPrefix(value, "*") {
		return nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v < min || v > max {
		return fmt.Errorf("must be between %d and %d", min, max)
	}
	return nil
}

// CreateCron creates a new cron job
func CreateCron() {
	var (
		minute         = "0"
		hour           string
		dayOfMonth     = "*"
		month          = "*"
		dayOfWeek      = "*"
		cronExpression string
		confirm        bool
	)

	fmt.Printf("%s=== CREATE CRON JOB ===%s\n\n", output.ColBlue, output.ColReset)
	logPath, _ := config.GetLogPath()

	fmt.Printf("%sInfo: Cron job will use:%s\n", output.ColBlue, output.ColReset)
	fmt.Printf("  URLs: %s\n", logPath)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Minute").Description("When in the hour? (0-59)").Placeholder("0").
				Validate(func(s string) error { return validateCronField(s, 0, 59, false) }).Value(&minute),
			huh.NewInput().Title("Hour").Description("When in the day? (0-23, */6 = every 6h)").Placeholder("8").
				Validate(func(s string) error { return validateCronField(s, 0, 23, true) }).Value(&hour),
		).WithTheme(huh.ThemeBase16()),
		huh.NewGroup(
			huh.NewInput().Title("Day of month").Description("Which day? (*) = every day").
				Validate(func(s string) error { return validateCronField(s, 1, 31, true) }).Value(&dayOfMonth),
			huh.NewSelect[string]().Title("Month").Description("Which month? (*) = every month").
				Options(
					huh.NewOption("Every month (*)", "*"),
					huh.NewOption("January (1)", "1"), huh.NewOption("February (2)", "2"),
					huh.NewOption("March (3)", "3"), huh.NewOption("April (4)", "4"),
					huh.NewOption("May (5)", "5"), huh.NewOption("June (6)", "6"),
					huh.NewOption("July (7)", "7"), huh.NewOption("August (8)", "8"),
					huh.NewOption("September (9)", "9"), huh.NewOption("October (10)", "10"),
					huh.NewOption("November (11)", "11"), huh.NewOption("December (12)", "12"),
				).Value(&month),
		).WithTheme(huh.ThemeBase16()),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Weekday").Description("Which day? (*) = every day").
				Options(
					huh.NewOption("Every day (*)", "*"),
					huh.NewOption("Sunday (0)", "0"), huh.NewOption("Monday (1)", "1"),
					huh.NewOption("Tuesday (2)", "2"), huh.NewOption("Wednesday (3)", "3"),
					huh.NewOption("Thursday (4)", "4"), huh.NewOption("Friday (5)", "5"),
					huh.NewOption("Saturday (6)", "6"),
				).Value(&dayOfWeek),
		).WithTheme(huh.ThemeBase16()),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Confirm").
				Description(fmt.Sprintf("Cron: %s %s %s %s %s\nCommand: %s -ci\nLog:  %s",
					minute, hour, dayOfMonth, month, dayOfWeek, getBinaryPathSafe(), logPath)).
				Value(&confirm),
		).WithTheme(huh.ThemeBase16()),
	)

	if err := form.Run(); err != nil {
		fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
		return
	}

	if !confirm {
		fmt.Println("Abort.")
		return
	}

	cronExpression = fmt.Sprintf("%s %s %s %s %s", minute, hour, dayOfMonth, month, dayOfWeek)
	binaryPath := getBinaryPathSafe()

	if err := installCronJob(binaryPath, cronExpression); err != nil {
		fmt.Printf("%sError: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}
}

func getBinaryPathSafe() string {
	binaryPath, err := getBinaryPath()
	if err != nil {
		return "./cert-checker"
	}
	return binaryPath
}

func getBinaryPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(ex)
}

func installCronJob(binaryPath, cronExpression string) error {
	logFile, err := config.GetLogPath()
	if err != nil {
		return fmt.Errorf("could not determine log path: %w", err)
	}

	comment := fmt.Sprintf("# cert-checker - %s", time.Now().Format(constants.CronDateFormat))
	cronEntry := fmt.Sprintf("%s %s -ci >> %s 2>&1", cronExpression, binaryPath, logFile)

	cmd := exec.Command("crontab", "-l")
	out, err := cmd.Output()
	currentCrontab := string(out)
	if err != nil {
		currentCrontab = ""
	}

	if strings.Contains(currentCrontab, binaryPath) && strings.Contains(currentCrontab, "-ci") {
		fmt.Printf("%sEntry already exists.%s\n", output.ColYellow, output.ColReset)
		fmt.Println("No changes made.")
		return nil
	}

	newCrontab := currentCrontab
	if newCrontab != "" && !strings.HasSuffix(newCrontab, "\n") {
		newCrontab += "\n"
	}
	newCrontab += comment + "\n" + cronEntry + "\n"

	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setting crontab failed: %w", err)
	}

	fmt.Printf("\n%sCron job created successfully!%s\n", output.ColGreen, output.ColReset)
	fmt.Printf("Cron-Expression: %s\n", cronExpression)
	fmt.Printf("Command: %s\n", cronEntry)
	fmt.Printf("%sLog: ./cert-checker -log OR tail -f %s%s\n", output.ColYellow, logFile, output.ColReset)
	return nil
}

// getCronJobs central function for parsing the crontab
func getCronJobs() ([]CronJob, error) {
	cmd := exec.Command("crontab", "-l")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("no crontab found")
	}

	lines := strings.Split(string(out), "\n")
	var jobs []CronJob

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, constants.CronMarker) {
			var command string
			if i+1 < len(lines) {
				command = strings.TrimSpace(lines[i+1])
			} else {
				command = "(Command not found)"
			}
			jobs = append(jobs, CronJob{
				Comment:   line,
				Command:   command,
				Index:     i,
				FullEntry: line + "\n" + command,
			})
			i++ // skip command line
		}
	}
	return jobs, nil
}

// ListAndManageJobs lists and manages cron jobs
func ListAndManageJobs() {
	fmt.Printf("%s=== CERTIFICATE CHECK CRON JOBS ===%s\n\n", output.ColBlue, output.ColReset)

	jobs, err := getCronJobs()
	if err != nil {
		fmt.Printf("%s%s%s\n", output.ColYellow, err.Error(), output.ColReset)
		fmt.Println("Create one with: ./cert-checker -cron")
		return
	}

	if len(jobs) == 0 {
		fmt.Printf("%sNo cron jobs with 'cert-checker' found.%s\n", output.ColYellow, output.ColReset)
		fmt.Println("Create one with: ./cert-checker -cron")
		return
	}

	fmt.Printf("%sTotal: %d job(s) found.%s\n\n", output.ColBlue, len(jobs), output.ColReset)
	for i, job := range jobs {
		fmt.Printf("%s%d.%s %s\n", output.ColGreen, i+1, output.ColReset, job.Comment)
		fmt.Printf("   %s\n", job.Command)
		fmt.Println()
	}

	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose action").
				Description("What do you want to do?").
				Options(
					huh.NewOption("Delete cron jobs", "delete"),
					huh.NewOption("Back", "back"),
				).
				Value(&action),
		).WithTheme(huh.ThemeBase16()),
	)

	if err := form.Run(); err != nil {
		fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
		return
	}

	if action == "back" {
		fmt.Println("Back to main menu...")
		return
	}

	if action == "delete" {
		var selectedJobs []int
		var confirm bool

		options := []huh.Option[int]{}
		for i, job := range jobs {
			shortDesc := strings.Replace(job.Comment, constants.CronMarker+" - ", "", 1)
			options = append(options, huh.NewOption(
				fmt.Sprintf("%d. %s | %s", i+1, shortDesc, output.TruncateString(job.Command, 40)),
				i,
			))
		}

		form = huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[int]().
					Title("Select jobs to delete").
					Description("Space to select, Enter to confirm").
					Limit(len(jobs)).
					Options(options...).
					Value(&selectedJobs),
			).WithTheme(huh.ThemeBase16()),
		)

		if err := form.Run(); err != nil {
			fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
			return
		}

		if len(selectedJobs) == 0 {
			fmt.Println("No jobs selected. Cancel.")
			return
		}

		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Delete selected jobs?").
					Description(fmt.Sprintf("%d job(s) will be deleted. Continue?", len(selectedJobs))).
					Value(&confirm),
			).WithTheme(huh.ThemeBase16()),
		)

		if err := confirmForm.Run(); err != nil {
			fmt.Printf("%sAbort.%s\n", output.ColRed, output.ColReset)
			return
		}

		if !confirm {
			fmt.Println("Cancel.")
			return
		}

		if err := removeSelectedCronJobs(selectedJobs); err != nil {
			fmt.Printf("%sError: %v%s\n", output.ColRed, err, output.ColReset)
			return
		}
	}
}

func removeSelectedCronJobs(selectedIndices []int) error {
	cmd := exec.Command("crontab", "-l")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("reading crontab failed: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	type Entry struct {
		Line     string
		ToDelete bool
	}

	entries := make([]Entry, len(lines))
	for i := range lines {
		entries[i] = Entry{Line: lines[i], ToDelete: false}
	}

	jobIndex := 0
	for i := 0; i < len(entries); i++ {
		line := entries[i].Line
		if strings.Contains(line, constants.CronMarker) {
			for _, selIdx := range selectedIndices {
				if selIdx == jobIndex {
					entries[i].ToDelete = true
					if i+1 < len(entries) {
						entries[i+1].ToDelete = true
					}
					break
				}
			}
			jobIndex++
		}
	}

	var newLines []string
	for _, entry := range entries {
		if !entry.ToDelete {
			newLines = append(newLines, entry.Line)
		}
	}

	newCrontab := strings.Join(newLines, "\n")
	if newCrontab != "" {
		newCrontab += "\n"
	}

	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("crontab update failed: %w", err)
	}

	fmt.Printf("\n%s%d cron job(s) removed successfully!%s\n", output.ColGreen, len(selectedIndices), output.ColReset)
	return nil
}

// ViewLogs
func ViewLogs() {
	logFile, err := config.GetLogPath()
	if err != nil {
		fmt.Printf("%sError getting log path: %v%s\n", output.ColRed, err, output.ColReset)
		return
	}

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Printf("%sNo log file found yet.%s\n", output.ColYellow, output.ColReset)
		fmt.Println("Logs are created when a Cron job runs.")
		fmt.Printf("Expected at: %s\n", logFile)
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return
	}

	cmd := exec.Command("less", "-R", logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("%sError opening pager: %v%s\n", output.ColRed, err, output.ColReset)
	}
}
