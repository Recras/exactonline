package odata2json

import (
	"testing"
)

func TestBinaryMarshalJSON_nil(t *testing.T) {
	b := Binary(nil)
	j, err := b.MarshalJSON()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if string(j) != "null" {
		t.Errorf("Expected json to be `null`, got %#v", j)
	}
}

func TestBinaryMarshalJSON_empty(t *testing.T) {
	b := Binary{}
	j, err := b.MarshalJSON()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if string(j) != `""` {
		t.Errorf("Expected json to be empty string, got %#v", j)
	}
}

func TestBinaryMarshalJSON_bytes(t *testing.T) {
	b := Binary("hello\n")
	j, err := b.MarshalJSON()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if string(j) != `"aGVsbG8K"` {
		t.Errorf("Expected b64(hello\\n) to be %#v, got %#v (%#v)", `"aGVsbG8K"`, j, string(j))
	}
}
