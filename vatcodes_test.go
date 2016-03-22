package exactonline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGetRecrasVATCodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/123/vat/VATCodes" {
			t.Errorf("Expected path to be correct, got %#v", r.URL.Path)
		}
		if f := r.URL.Query().Get("$filter"); f != "substringof('recras:', Description) eq true" {
			t.Errorf("Expected $filter to test substring 'recras:', got %#v", f)
		}

		fmt.Fprint(w, `{"d":{"results":[{"ID":"guid1","Code":"r1","Description":"recras:6"},{"ID":"guid2","Code":"r2","Description":"recras:21"}]}}`)
	}))
	defer ts.Close()
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	items, err := cl.GetRecrasVATCodes()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	} else if (items[0] != VATCode{ID: "guid1", Code: "r1", Description: "recras:6"}) {
		t.Errorf("Expected first item, got %#v", items[0])
	}
}

func TestGetRecrasVATCodes_NoDivision(t *testing.T) {
	cl := Client{}
	_, err := cl.GetRecrasVATCodes()
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}
