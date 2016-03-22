package odata2json

import (
	"testing"
	"time"
)

func TestMarshalJSON(t *testing.T) {
	d := &Date{Time: time.Date(2015, 1, 2, 0, 0, 0, 0, time.UTC)}
	b, err := d.MarshalJSON()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if string(b) != `"2015-01-02"` {
		t.Errorf("Expected date to be `\"2015-01-02\"`, got `%s`", string(b))
	}
}

func TestUnmarshalJSON_NoOffset(t *testing.T) {
	d := &Date{}
	if err := d.UnmarshalJSON([]byte(`"\/Date(1234)\/"`)); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	} else if d.Time != time.Unix(1, 234e6) {
		t.Errorf("Expected time to be %#v, got %#v", time.Unix(1, 234e6), d.Time)
	}
}
func TestUnmarshalJSON_PostiveOffset(t *testing.T) {
	d := &Date{}
	if err := d.UnmarshalJSON([]byte(`"\/Date(1234+1)\/"`)); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	} else if d.Time != time.Unix(61, 234e6) {
		t.Errorf("Expected time to be %#v, got %#v", time.Unix(61, 234e6), d.Time)
	}
}
func TestUnmarshalJSON_NegativeOffset(t *testing.T) {
	d := &Date{}
	if err := d.UnmarshalJSON([]byte(`"\/Date(61234-1)\/"`)); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	} else if d.Time != time.Unix(1, 234e6) {
		t.Errorf("Expected time to be %#v, got %#v", time.Unix(1, 234e6), d.Time)
	}
}

func TestUnmarshalJSONIncorrectFormat(t *testing.T) {
	d := &Date{}
	if err := d.UnmarshalJSON([]byte(`"1234)/"`)); err == nil {
		t.Errorf("Expected an error if `/Date(` is missing")
	}
	if err := d.UnmarshalJSON([]byte(`"\/Date(123"`)); err == nil {
		t.Errorf("Expected an error if `)/` is missing")
	}
}
