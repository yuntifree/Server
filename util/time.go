package util

import "time"

//GetNextCqssc return next cqssc time
func GetNextCqssc(tt time.Time) time.Time {
	year, month, day := tt.Date()
	local := tt.Location()
	hour, min, _ := tt.Clock()
	if hour >= 10 && hour < 22 {
		min = (min/10 + 1) * 10
	} else if hour >= 22 && hour < 2 {
		min = (min/5 + 1) * 5
	} else {
		hour = 10
		min = 0
	}

	return time.Date(year, month, day, hour, min, 0, 0, local)
}

//TruncTime  trunc time for interval minutes
func TruncTime(tt time.Time, interval int) time.Time {
	if interval == 0 {
		return tt
	}
	year, month, day := tt.Date()
	local := tt.Location()
	hour, min, _ := tt.Clock()
	min = (min / interval) * interval
	return time.Date(year, month, day, hour, min, 0, 0, local)
}
