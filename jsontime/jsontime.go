package jsontime

import (
	"errors"
	"time"
)

const TimeFormat = "2006-01-02T15:04:05-0700"

type Time struct {
	time time.Time
}

func Now() Time {
	return Time{time.Now()}
}

func (t Time) MarshalJSON() ([]byte, error) {
	stamp := `"` + t.time.Format(TimeFormat) + `"`
	return []byte(stamp), nil
}

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return errors.New("invalid jsontime")
	}
	if nativeTime, err := time.Parse(TimeFormat, string(data[1:len(data)-1])); err == nil {
		t.time = nativeTime
	}
	return
}
func (t Time) GetTime() time.Time {
	return t.time
}
func (t Time) Unix() int64 {
	return t.time.Unix()
}

func (t Time) IsZero() bool {
	return t.time.IsZero()
}

func (t Time) Add(d time.Duration) Time {
	return Time{time: t.time.Add(d)}
}
func (t Time) After(u Time) bool {
	return t.time.After(u.GetTime())
}

func (t Time) Before(u Time) bool {
	return t.time.Before(u.GetTime())
}
