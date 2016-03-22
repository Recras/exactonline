package recras

import (
	"encoding/json"
	"testing"
)

func TestNewPersoneel(t *testing.T) {
	_ = Personeel{
		ID:               1,
		Displaynaam:      "Test",
		ContactpersoonID: 2,
	}
}

func TestMarshalJson(t *testing.T) {
	p := Personeel{
		ID:               1,
		Displaynaam:      "Test",
		ContactpersoonID: 2,
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}

	if ref := `{"id":1,"displaynaam":"Test","contactpersoon_id":2}`; string(b) != ref {
		t.Errorf("Expected MarshalJSON to return %#v, got %#v", ref, string(b))
	}
}
