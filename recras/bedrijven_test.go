package recras

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestGetBedrijven(t *testing.T) {
	ts, c := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id": 1, "bedrijfsnaam": "asf", "btw_nummer": "NL123456789B01"}]`)
	})
	defer ts.Close()

	bs, err := c.GetBedrijven(nil)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(bs) != 1 {
		t.Errorf("Expected number of Facturen to be 1, got %d", len(bs))
	}
	compare := Bedrijf{
		ID:           1,
		Bedrijfsnaam: "asf",
		BTWNummer:    "NL123456789B01",
	}
	if !reflect.DeepEqual(bs[0], compare) {
		t.Errorf("Expected %v, got %v", compare, bs[0])
	}
}
