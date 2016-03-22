package exactonline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestFindAccountByRecrasID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; p != "/api/v1/123/crm/Accounts" {
			t.Errorf("Expected path to be `/api/v1/123/crm/Accounts`, got `%s`", p)
		}
		if f := r.URL.Query().Get("$filter"); f != "SearchCode eq 'K12'" {
			t.Errorf("Expected $filter to be `SearchCode eq 'K12'`, got `%s`", f)
		}
		fmt.Fprint(w, `{"d":{"results":[{"Code": "                12", "ID": "guid", "SearchCode": "K12"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	item, err := cl.FindAccountByRecrasID(12)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if (item != Account{Code: "                12", ID: "guid", SearchCode: "K12"}) {
		t.Errorf("Expected the item, got %#v", item)
	}
}
func TestFindAccountByRecrasID_NotFound(t *testing.T) {
	findBySearchCodeCalled := false
	findByCodeCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; p != "/api/v1/123/crm/Accounts" {
			t.Errorf("Expected path to be `/api/v1/123/logistics/Accounts`, got `%s`", p)
		}
		if f := r.URL.Query().Get("$filter"); f == "SearchCode eq 'K12'" {
			findBySearchCodeCalled = true
		} else if f == "Code eq '732727000000000012'" {
			findByCodeCalled = true
		}
		fmt.Fprint(w, `{"d":{"results":[]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	_, err := cl.FindAccountByRecrasID(12)
	if !findBySearchCodeCalled {
		t.Errorf("Expected Accounts API to be queried by SearchCode")
	}
	if !findByCodeCalled {
		t.Errorf("Expected Accounts API to be queried by Code")
	}
	if err == nil {
		t.Errorf("Expected error")
	} else if _, ok := err.(ErrAccountNotFound); !ok {
		t.Errorf("Expected ErrAccountNotFound, got %#v", err)
	}
}

func TestFindAccountByRecrasID_FoundByOldId(t *testing.T) {
	findBySearchCodeCalled := false
	findByCodeCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; p != "/api/v1/123/crm/Accounts" {
			t.Errorf("Expected path to be `/api/v1/123/logistics/Accounts`, got `%s`", p)
		}
		if f := r.URL.Query().Get("$filter"); f == "SearchCode eq 'K12'" {
			findBySearchCodeCalled = true
			fmt.Fprintf(w, `{"d":{"results":[]}}`)
			return
		}
		if f := r.URL.Query().Get("$filter"); f == "Code eq '732727000000000012'" {
			findByCodeCalled = true
			fmt.Fprintf(w, `{"d":{"results":[{"Code": "                12", "ID": "guid", "SearchCode": null}]}}`)
		}
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	item, err := cl.FindAccountByRecrasID(12)
	if !findBySearchCodeCalled {
		t.Errorf("Expected Accounts API to be queried by SearchCode")
	}
	if !findByCodeCalled {
		t.Errorf("Expected Accounts API to be queried by Code")
	}
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if (item != Account{Code: "                12", ID: "guid", SearchCode: ""}) {
		t.Errorf("Expected the item, got %#v", item)
	}
}

func TestFindAccountByRecrasID_NoDivision(t *testing.T) {
	cl := Client{}
	_, err := cl.FindAccountByRecrasID(12)
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestSave(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected method to be `POST`, got `%s`", r.Method)
		}
		if r.URL.Path != fmt.Sprintf(accountURI, 123) {
			t.Errorf("Expected path to be set, got %#v", r.URL.Path)
		}
		if h := r.Header.Get("Content-Type"); h != "application/json" {
			t.Errorf("Expected content-type header to be application/json")
		}

		payload := map[string]interface{}{}
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&payload)
		if err != nil {
			panic("Error decoding json, should never happen, error: " + err.Error())
		}

		payload["ID"] = "guid"

		out := map[string]interface{}{
			"d": payload,
		}
		enc := json.NewEncoder(w)
		_ = enc.Encode(&out)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123

	a := Account{}
	if err := a.Save(cl); err != ErrAccountNameRequired {
		t.Errorf("Expected ErrAccountNameRequired if Name is missing, got %#v", err)
	}

	a.Name = "Blabla"
	if err := a.Save(cl); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if a.ID != "guid" {
		t.Errorf("Expected ID to be set after save")
	}
}

func TestSaveAccount_NoDivision(t *testing.T) {
	cl := &Client{}
	a := Account{Name: "asdf"}
	err := a.Save(cl)
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}
