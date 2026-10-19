package quota

import (
	"time"
)

// CalculateWindow determines the [start, end) time range for a given window type and reference time.
func CalculateWindow(wType WindowType, durationSec int64, now time.Time) (time.Time, time.Time) {
	u := now.UTC()
	switch wType {
	case WindowTypeDaily:
		start := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 0, 1)
		return start, end
	case WindowTypeWeekly:
		// Monday as the start of week
		weekday := int(u.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday is day 7
		}
		daysToMonday := weekday - 1
		startDay := u.AddDate(0, 0, -daysToMonday)
		start := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 0, 7)
		return start, end
	case WindowTypeMonthly:
		start := time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		return start, end
	case WindowTypeDuration:
		if durationSec <= 0 {
			durationSec = 3600 // Default to 1 hour
		}
		bucket := (u.Unix() / durationSec) * durationSec
		start := time.Unix(bucket, 0).UTC()
		end := start.Add(time.Duration(durationSec) * time.Second)
		return start, end
	default:
		// Fallback to daily
		start := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 0, 1)
		return start, end
	}
}
