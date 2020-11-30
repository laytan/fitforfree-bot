package times

import (
	"time"
)

const FullLayout = "15:04 02-01-2006"
const DateLayout = "02-01-2006"
const TimeLayout = "15:04"

func FormatTimestamp(timestamp uint, layout string) string {
	t := time.Unix(int64(timestamp), 0)
	return t.Format(layout)
}

func FromInput(input string, layout string) (time.Time, error) {
	return time.Parse(layout, input)
}
