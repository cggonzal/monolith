package jobs

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// nextCronTime returns the next time after 'from' that matches the provided
// cron expression. Only the minute and hour fields are honoured. The other
// fields must be '*'. Supported forms for minute and hour are:
//   - "*"         -> every minute/hour
//   - "*/N"       -> every N minutes/hours
//   - "M" or "H"  -> a specific minute or hour
func nextCronTime(expr string, from time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, errors.New("invalid cron expression")
	}
	minuteField := fields[0]
	hourField := fields[1]
	if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
		return time.Time{}, errors.New("unsupported cron expression")
	}
	t := from.Add(time.Minute).Truncate(time.Minute)
	for i := 0; i < 525600; i++ { // search up to one year
		if matchCronField(minuteField, t.Minute()) && matchCronField(hourField, t.Hour()) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, errors.New("unable to compute next run time")
}

func matchCronField(field string, v int) bool {
	if field == "*" {
		return true
	}
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(field[2:])
		if err != nil || n == 0 {
			return false
		}
		return v%n == 0
	}
	val, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return v == val
}
