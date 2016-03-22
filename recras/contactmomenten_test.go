package recras

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func newContactmoment() Contactmoment {
	d := time.Date(2014, 7, 11, 11, 0, 0, 0, time.UTC)
	return Contactmoment{
		ID:               1,
		ContactID:        2,
		ContactpersoonID: 3,
		Onderwerp:        "Onderwerp",
		Bericht:          "Bericht",
		Ondertekening:    "superdoei",
		Sticky:           true,
		ContactOpnemen:   &d,
		Soort:            "noot",
	}
}

func TestMarshalJSON(t *testing.T) {
	c := newContactmoment()
	bb, _ := json.Marshal(c)
	var res Contactmoment
	_ = json.Unmarshal(bb, &res)
	if *res.ContactOpnemen != *c.ContactOpnemen {
		t.Errorf("Expected ContactOpnemen to be %#v, got %#v", c.ContactOpnemen, res.ContactOpnemen)
	}
	res.ContactOpnemen = nil
	c.ContactOpnemen = nil
	if res != c {
		t.Logf("%s %s", c.ContactOpnemen, res.ContactOpnemen)
		t.Errorf("Expected result to be %#v, got %#v", c, res)
	}
}

func TestMarshalJSON_NoContactOpnemen(t *testing.T) {
	c := newContactmoment()
	c.ContactOpnemen = nil
	bb, _ := json.Marshal(c)
	if !strings.Contains(string(bb), `"contact_opnemen":null`) {
		t.Errorf("Expected contact_opnemen to be null")
	}
}

func TestSaveContactmoment(t *testing.T) {
	apiCalled := false
	ts, client := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		dec := json.NewDecoder(r.Body)
		m := map[string]interface{}{}
		dec.Decode(&m)
		if _, ok := m["id"]; ok {
			t.Errorf("Expected id to be omitted")
		}
		m["id"] = 1337
		enc := json.NewEncoder(w)
		enc.Encode(m)
	})
	defer ts.Close()
	c := newContactmoment()
	c.ID = 0
	c.Save(client)
	if !apiCalled {
		t.Errorf("Expected API to be called")
	}
	if c.ID != 1337 {
		t.Errorf("Expected ID to be set")
	}
}

func TestSaveContactmoment_NoSoort(t *testing.T) {
	c := newContactmoment()
	c.Soort = ""
	err := c.Save(nil)
	if err != ErrNoSoort {
		t.Errorf("Expected ErrNoSoort, got %#v", err)
	}
}
