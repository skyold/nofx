package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// UnixTime wraps int64 to support scanning from string timestamps (e.g. from SQLite)
// and marshaling to int64 for JSON (Unix milliseconds).
type UnixTime int64

// Scan implements sql.Scanner
func (t *UnixTime) Scan(value interface{}) error {
	if value == nil {
		*t = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*t = UnixTime(v)
	case int:
		*t = UnixTime(v)
	case float64:
		*t = UnixTime(int64(v))
	case []byte:
		return t.scanString(string(v))
	case string:
		return t.scanString(v)
	case time.Time:
		*t = UnixTime(v.UnixMilli())
	default:
		return fmt.Errorf("cannot scan type %T into UnixTime", value)
	}
	return nil
}

func (t *UnixTime) scanString(s string) error {
	// Try parsing as integer first
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		*t = UnixTime(i)
		return nil
	}

	// Try parsing as timestamp string
	// Common formats:
	// "2026-01-07 00:28:08.158+00:00" (Postgres/SQLite default)
	// "2006-01-02 15:04:05"
	// "2006-01-02T15:04:05Z"
	formats := []string{
		"2006-01-02 15:04:05.999-07:00",
		"2006-01-02 15:04:05.999Z07:00",
		"2006-01-02 15:04:05-07:00",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, s); err == nil {
			*t = UnixTime(parsed.UnixMilli())
			return nil
		}
	}

	return fmt.Errorf("could not parse time string: %s", s)
}

// Value implements driver.Valuer
func (t UnixTime) Value() (driver.Value, error) {
	return int64(t), nil
}

// MarshalJSON implements json.Marshaler
func (t UnixTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(t))
}

// UnmarshalJSON implements json.Unmarshaler
func (t *UnixTime) UnmarshalJSON(data []byte) error {
	var i int64
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	*t = UnixTime(i)
	return nil
}

// ToTime returns time.Time
func (t UnixTime) ToTime() time.Time {
	return time.UnixMilli(int64(t))
}
