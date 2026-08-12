package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type DurationOrInt string

func (d *DurationOrInt) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*d = DurationOrInt(s)
		return nil
	}
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		*d = DurationOrInt(strconv.Itoa(i))
		return nil
	}
	return fmt.Errorf("invalid duration or int: %s", string(b))
}

func parseDurationOrInt(val string) (int, error) {
	d, err := time.ParseDuration(val)
	if err == nil {
		return int(d.Seconds()), nil
	}
	i, err := strconv.Atoi(val)
	if err == nil {
		return i, nil
	}
	return 0, fmt.Errorf("cannot parse %q as duration or int", val)
}
