package exactonline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestJournal(t *testing.T) {
	_ = Journal{
		ID:          "guid",
		Code:        "test",
		Description: "Asdf",
	}
}

func TestFindDefaultJournal_Found(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"d":{"results":[{"ID":"guid","Code":"recras","Description":"asdf"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234
	j, err := cl.FindDefaultJournal()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	ref := Journal{
		ID:          "guid",
		Code:        "recras",
		Description: "asdf",
	}
	if j != ref {
		t.Errorf("Expected journal got %#v", j)
	}
}

func TestFindDefaultJournal_NotFound(t *testing.T) {
	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/financial/Journals" {
			apiCalled = true
			if f := r.URL.Query().Get("$filter"); f != "Code eq 'recras'" {
				t.Errorf("Expected call to filter on Code='recras', got %#v", f)
			}
		} else {
			t.Errorf("Expected path to be %s, got %s", "/api/v1/1234/financial/Journals", r.URL.Path)
		}
		fmt.Fprint(w, `{"d":{"results":[]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234
	_, err := cl.FindDefaultJournal()
	if !apiCalled {
		t.Errorf("Expected Journals API to be called")
	}
	if err != ErrJournalNotFound {
		t.Errorf("Expected ErrJournalNotFound, got %#v", err)
	}
}
