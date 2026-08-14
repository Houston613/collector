package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type DurationOrInt int

func parseDurationOrInt(s string) (int, error) {
	dur, err := time.ParseDuration(s)
	if err == nil {
		return int(dur.Seconds()), nil
	}
	num, err := strconv.Atoi(s)
	if err == nil {
		return num, nil
	}
	return 0, fmt.Errorf("cannot parse %q as duration or int: %w", s, err)
}

func (d *DurationOrInt) UnmarshalJSON(b []byte) error {
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		*d = DurationOrInt(i)
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		val, err := parseDurationOrInt(s)
		if err != nil {
			return err
		}
		*d = DurationOrInt(val)
		return nil
	}

	return fmt.Errorf("invalid duration or int: %s", string(b))
}

