package odata2json

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var ErrUnmarshalDate = errors.New("Error unmarshaling date")
var odataDateRegex = regexp.MustCompile(`^\/Date\(([0-9]+)([+-][0-9]+)?\)\/$`)

const marshalFormat = "\"2006-01-02\""

type Date struct {
	Time time.Time
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}
	matches := odataDateRegex.FindStringSubmatch(str)
	if len(matches) <= 1 {
		return ErrUnmarshalDate
	}
	var minutes, millis int64

	fmt.Sscan(string(matches[1]), &millis)
	if len(matches) == 3 {
		fmt.Sscan(string(matches[2]), &minutes)
	}

	d.Time = time.Unix(millis/1000+minutes*60, millis%1000*1e6)
	return nil
}

func (d *Date) MarshalJSON() ([]byte, error) {
	return []byte(d.Time.Format(marshalFormat)), nil
}
