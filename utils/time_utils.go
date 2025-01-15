package utils

import "time"

func StartDay(t time.Time) time.Time {
	return t.Add(8 * time.Hour).Truncate(24 * time.Hour).Add(-8 * time.Hour)
}

func EndDay(t time.Time) time.Time {
	return t.Add(8 * time.Hour).Truncate(24 * time.Hour).Add(16 * time.Hour)
}

func StartWeek(t time.Time) time.Time {
	return t.Add(8*time.Hour).Truncate(24*time.Hour).AddDate(0, 0, -int(time.Now().Weekday())).Add(-8 * time.Hour)
}

func EndWeek(t time.Time) time.Time {
	return t.Add(8*time.Hour).Truncate(24*time.Hour).AddDate(0, 0, 6-int(time.Now().Weekday())).Add(-8 * time.Hour)
}
