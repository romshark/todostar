package timefmt

import (
	"fmt"
	"time"
)

const TimeFormat = "2006-01-02T15:04"

func Due(now, due time.Time) string {
	d := due.Sub(now)
	if d > 0 && d < time.Minute {
		return "due in a moment"
	}
	// Overdue
	if d > -60*time.Second && d < 0 {
		return "due now"
	} else if d < 0 {
		return fmt.Sprintf("Over due by %s", Dur(-d))
	}
	return fmt.Sprintf("due in %s", Dur(d))
}

func Dur(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

func DateTimeStr(tm time.Time) string {
	if tm.IsZero() {
		return ""
	}
	return tm.Format(TimeFormat)
}
