// internal/schedule/schedule.go
package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cert-checker/internal/output"

	"github.com/charmbracelet/huh"
)

var (
	minute     string = "0"
	hour       string
	dayOfMonth string = "*"
	month      string = "*"
	dayOfWeek  string = "*"
)

// cron job entry with comment and command
type CronJob struct {
	Comment   string
	Command   string
	Index     int    // position in the crontab (for deletion)
	FullEntry string // original line (comment + command)
}

// Schedule launches the interactive menu for cron job setup
func ScheduleMain() {
	var action string

	fmt.Printf("%s=== CRON JOB SETUP ===%s\n", output.ColBlue, output.ColReset)
	fmt.Printf("%sNote: This job automatically checks the URLs from your configuration file.%s\n", output.ColYellow, output.ColReset)
	fmt.Printf("%sDefault path: ~/.config/cert-checker/urls.txt%s\n\n", output.ColYellow, output.ColReset)
	// cron menu
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

// create cron jobs
func CreateCron() {
	var cronExpression string
	var confirm bool

	fmt.Printf("%s=== CREATE CRON JOB ===%s\n\n", output.ColBlue, output.ColReset)
	// create cron
	form := huh.NewForm(
		// minute (0-59)
		huh.NewGroup(
			huh.NewInput().
				Title("Minute").
				Description("When in the hour? (0-59)").
				Placeholder("0").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("Please enter a minute between 0-59")
					}
					m, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("Please enter a number")
					}
					if m < 0 || m > 59 {
						return fmt.Errorf("0 59")
					}
					return nil
				}).
				Value(&minute),

			// hour of the day
			huh.NewInput().
				Title("Hour").
				Description("When in the day? (0-23, 0=midnight, */6 = every 6 hours)").
				Placeholder("8").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("Please enter an hour")
					}
					if strings.HasPrefix(s, "*/") {
						// cron expressions like */6
						return nil
					}
					h, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("Please enter a number or */N")
					}
					if h < 0 || h > 23 {
						return fmt.Errorf("Hour must be between 0 and 23")
					}
					return nil
				}).
				Value(&hour),
		).WithTheme(huh.ThemeBase16()),

		// day of the month (1-31)
		huh.NewGroup(
			huh.NewInput().
				Title("Day of the month").
				Description("Which day? (*) = every day").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("Enter day")
					}
					if strings.HasPrefix(s, "*") {
						return nil
					}
					d, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("Please enter a number")
					}
					if d < 0 || d > 31 {
						return fmt.Errorf("1 31")
					}
					return nil
				}).
				Value(&dayOfMonth),

			// month (1-12) or "*" for each month
			huh.NewSelect[string]().
				Title("Month").
				Description("Which month? (*) = every month").
				Options(
					huh.NewOption("Every month (*)", "*"),
					huh.NewOption("January (1)", "1"),
					huh.NewOption("February (2)", "2"),
					huh.NewOption("March (3)", "3"),
					huh.NewOption("April (4)", "4"),
					huh.NewOption("May (5)", "5"),
					huh.NewOption("June (6)", "6"),
					huh.NewOption("July (7)", "7"),
					huh.NewOption("August (8)", "8"),
					huh.NewOption("September (9)", "9"),
					huh.NewOption("October (10)", "10"),
					huh.NewOption("November (11)", "11"),
					huh.NewOption("December (12)", "12"),
				).
				Value(&month),
		).WithTheme(huh.ThemeBase16()),

		// day of the week (0-6, 0=Sunday) or "*" for every day
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Weekday").
				Description("Which day of the week? (*) = every day").
				Options(
					huh.NewOption("Every day (*)", "*"),
					huh.NewOption("Sunday (0)", "0"),
					huh.NewOption("Montag (1)", "1"),
					huh.NewOption("Tuesday (2)", "2"),
					huh.NewOption("Wednesday (3)", "3"),
					huh.NewOption("Thursday (4)", "4"),
					huh.NewOption("Friday (5)", "5"),
					huh.NewOption("Saturday (6)", "6"),
				).
				Value(&dayOfWeek),
		).WithTheme(huh.ThemeBase16()),

		// confirm
		huh.NewGroup(
			huh.NewConfirm().
				Title("Confirm").
				Description(fmt.Sprintf("Cron: %s %s %s %s %s\nCommand: %s -ci",
					minute, hour, dayOfMonth, month, dayOfWeek, getBinaryPathSafe())).
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

	// create cron job
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
	comment := fmt.Sprintf("# cert-checker - %s", time.Now().Format("2006-01-02 15:04:05"))

	cronEntry := fmt.Sprintf("%s %s -ci >> /tmp/cert-check.log 2>&1", cronExpression, binaryPath)

	// get current crontab
	cmd := exec.Command("crontab", "-l")
	out, err := cmd.Output()
	currentCrontab := string(out)

	if err != nil {
		currentCrontab = ""
	}

	// Check whether entry already exists
	if strings.Contains(currentCrontab, binaryPath) && strings.Contains(currentCrontab, "-ci") {
		fmt.Printf("%sEntry already exists.%s\n", output.ColYellow, output.ColReset)
		fmt.Println("No changes made.")
		return nil
	}

	// add new entry
	newCrontab := currentCrontab
	if newCrontab != "" && !strings.HasSuffix(newCrontab, "\n") {
		newCrontab += "\n"
	}
	newCrontab += comment + "\n" + cronEntry + "\n"

	// Crontab setzen
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Setting crontab failed: %w", err)
	}

	fmt.Printf("\n%sCron job created successfully!%s\n", output.ColGreen, output.ColReset)
	fmt.Printf("Cron-Expression: %s\n", cronExpression)
	fmt.Printf("command: %s\n", cronEntry)
	fmt.Print("TEST: ", comment)
	fmt.Printf("Log: %stail -f /tmp/cert-check.log%s\n", output.ColBlue, output.ColReset)
	return nil
}

// ListAndManageJobs
func ListAndManageJobs() {
	fmt.Printf("%s=== CERTIFICATE CHECK CRON JOBS ===%s\n\n", output.ColBlue, output.ColReset)

	// read crontab
	cmd := exec.Command("crontab", "-l")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("%sNo crontab found or empty.%s\n", output.ColYellow, output.ColReset)
		return
	}

	lines := strings.Split(string(out), "\n")
	jobs := []CronJob{}
	jobCount := 0

	// collect jobs
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// search for the comment
		if strings.Contains(line, "# cert-checker") {
			jobCount++

			// search for the command
			var command string
			if i+1 < len(lines) {
				command = strings.TrimSpace(lines[i+1])
			} else {
				command = "(Command not found)"
			}

			// save job
			jobs = append(jobs, CronJob{
				Comment:   line,
				Command:   command,
				Index:     i,
				FullEntry: line + "\n" + command,
			})

			// Increase index to skip command line
			i++
		}
	}

	if len(jobs) == 0 {
		fmt.Printf("%sNo cron jobs with 'cert-checker' found.%s\n", output.ColYellow, output.ColReset)
		fmt.Println("Create one with: ./cert-checker -i")
		return
	}

	// print jobs
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
			shortDesc := strings.Replace(job.Comment, "# cert-checker - ", "", 1)
			options = append(options, huh.NewOption(
				fmt.Sprintf("%d. %s | %s", i+1, shortDesc, truncate(job.Command, 40)),
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
			fmt.Println("No jobs selected. Abbruch.")
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
			fmt.Println("Abort.")
			return
		}

		if err := removeSelectedCronJobs(selectedJobs); err != nil {
			fmt.Printf("%sError: %v%s\n", output.ColRed, err, output.ColReset)
			return
		}
	}
}

// removeSelectedCronJobs deletes selected cron jobs
func removeSelectedCronJobs(selectedIndices []int) error {
	cmd := exec.Command("crontab", "-l")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("reading crontab failed: %w", err)
	}

	originalCrontab := string(out)
	lines := strings.Split(originalCrontab, "\n")

	type Entry struct {
		Line     string
		ToDelete bool
	}

	entries := make([]Entry, len(lines))
	for i, line := range lines {
		entries[i] = Entry{Line: line, ToDelete: false}
	}

	jobIndex := 0
	for i := 0; i < len(entries); i++ {
		line := entries[i].Line
		if strings.Contains(line, "# cert-checker") {
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
	newCrontab = strings.TrimRight(newCrontab, "\n")
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

// truncate cuts long strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
