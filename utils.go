package main

import (
	"regexp"
	"strings"
	"time"
)

type void struct{}

var dateRegexp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

func ConvertRelativeToAbsoluteTime(relativeTime string, date time.Time) string {
	nowDate := date.Format("2006-01-02")
	nextDate := date.AddDate(0, 0, 1).Format("2006-01-02")
	dayAfterNextDate := date.AddDate(0, 0, 2).Format("2006-01-02")

	replacer := strings.NewReplacer(
		"今天", nowDate,
		"明天", nextDate,
		"後天", dayAfterNextDate,
	)

	return replacer.Replace(relativeTime)
}

func HasStatusIsOld(status string, referenceDate time.Time) bool {
	referenceDate = referenceDate.Truncate(24 * time.Hour)

	for _, match := range dateRegexp.FindAllString(status, -1) {
		parsedDate, err := time.Parse("2006-01-02", match)
		if err != nil {
			continue
		}
		if !parsedDate.Before(referenceDate) {
			return false
		}
	}

	return true
}
