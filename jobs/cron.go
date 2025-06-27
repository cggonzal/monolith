package jobs

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// matcher represents a single cron field matcher.
type matcher func(int) bool

// parseField parses a cron field value and returns a matcher for that field.
// Supported forms:
//
//	"*"     - any value
//	"*/N"   - every N units
//	"N"     - exact value N
func parseField(field string, min, max int, dow bool) (matcher, error) {
	if field == "*" {
		return func(int) bool { return true }, nil
	}
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(field[2:])
		if err != nil || n <= 0 {
			return nil, errors.New("invalid step")
		}
		return func(v int) bool { return (v-min)%n == 0 }, nil
	}
	val, err := strconv.Atoi(field)
	if err != nil {
		return nil, errors.New("invalid field")
	}
	if dow && val == 7 {
		val = 0
	}
	if val < min || val > max {
		return nil, errors.New("value out of range")
	}
	return func(v int) bool { return v == val }, nil
}

// nextCronTime returns the next time after 'from' that matches the cron
// expression. It supports standard 5-field cron syntax (minute, hour,
// day-of-month, month, day-of-week) with simple forms for each field.
func nextCronTime(expr string, from time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, errors.New("invalid cron expression")
	}

	minuteM, err := parseField(fields[0], 0, 59, false)
	if err != nil {
		return time.Time{}, err
	}
	hourM, err := parseField(fields[1], 0, 23, false)
	if err != nil {
		return time.Time{}, err
	}
	domM, err := parseField(fields[2], 1, 31, false)
	if err != nil {
		return time.Time{}, err
	}
	monthM, err := parseField(fields[3], 1, 12, false)
	if err != nil {
		return time.Time{}, err
	}
	dowM, err := parseField(fields[4], 0, 7, true)
	if err != nil {
		return time.Time{}, err
	}

	t := from.Add(time.Minute).Truncate(time.Minute)
	// search up to two years ahead
	for i := 0; i < 2*525600; i++ {
		if minuteM(t.Minute()) && hourM(t.Hour()) &&
			monthM(int(t.Month())) &&
			(domM(t.Day()) || dowM(int(t.Weekday()))) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, errors.New("unable to compute next run time")
}
