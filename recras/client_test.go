package recras

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/Recras/exactonline/httperror"
)

func TestNewClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok {
			t.Errorf("Expected request to have basic authentication")
		} else {
			if u != "user" {
				t.Errorf("Expected basic auth user to be `user`, got `%s`", u)
			}
			if p != "password" {
				t.Errorf("Expected basic auth password to be `password`, got `%s`", p)
			}
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cl := NewClient("http://"+u.Host, "user", "password")
	if _, err := cl.Client.Get(ts.URL); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
}

func TestRoundTripBaseURL(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/foo" {
			t.Errorf("Expected path to be `/foo`, got %#v", r.URL.Path)
		}
	}))
	defer ts.Close()

	url, _ := url.Parse(ts.URL)
	c := &http.Client{Transport: &Transport{BaseURL: url}}
	c.Get("/foo")

	if !called {
		t.Errorf("Expected mock server to be called")
	}
}

type testBaseTransport struct{}

func (t *testBaseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := cloneRequest(req)
	req2.Header.Add("x-foo", "bar")

	res, err := http.DefaultTransport.RoundTrip(req2)
	return res, err
}

func TestRoundTripBaseTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-foo") != "bar" {
			t.Errorf("Expected X-FOO header to be `bar`")
		}
	}))
	defer ts.Close()

	c := &http.Client{Transport: &Transport{Base: &testBaseTransport{}}}
	c.Get(ts.URL)
}

func TestRoundTripBasicAuthOutsideDomain(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); ok {
			t.Errorf("Expected no basic auth, got (%s, %s)", u, p)
		}
	}))
	defer ts.Close()

	c := &http.Client{Transport: &Transport{BasicAuth: basicAuth{"user", "password"}}}
	c.Get(ts.URL)
}

func createTestAPI(fun func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *Client) {
	ts := httptest.NewServer(http.HandlerFunc(fun))

	u, _ := url.Parse(ts.URL)
	c := &Client{Client: http.Client{
		Transport: &Transport{
			BaseURL: u,
		},
	}}

	return ts, c
}

func TestGet(t *testing.T) {
	apiCalled := false
	item := map[string]interface{}{
		"Asdf": "HJKL",
		"Test": float64(1),
	}
	apiURL := "/asdf"
	ts, c := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true

		if r.URL.Path != apiURL {
			t.Errorf("Expected URL to be %#v, got %#v", apiURL, r.URL.Path)
		}

		enc := json.NewEncoder(w)
		enc.Encode(item)
	})
	defer ts.Close()

	i := map[string]interface{}{}
	err := c.Get(apiURL, &i)
	if err != nil {
		t.Fatalf("Expected no error, got %#v", err)
	}
	if !apiCalled {
		t.Errorf("Expected API to be called")
	}
	if !reflect.DeepEqual(i, item) {
		t.Errorf("Expected returned i to be %#v, got %#v", item, i)
	}
}

func TestGet_StatusCodeNotOK(t *testing.T) {
	ts, c := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	defer ts.Close()

	err := c.Get("", nil)
	if _, ok := err.(httperror.HTTPError); !ok {
		t.Fatalf("Expected HTTPError, got %#v", err)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	ts, c := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{")
	})
	defer ts.Close()

	err := c.Get("", nil)
	if err == nil {
		t.Fatalf("Expected error")
	}
}

func TestPost(t *testing.T) {
	apiCalled := false
	ts, c := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		if r.Method != "POST" {
			t.Errorf("Expected method to be POST, got %#v", r.Method)
		}
		if h := r.Header.Get("content-type"); h != "application/json" {
			t.Errorf("Expected content-type to be %#v, got %#v", "application/json", h)
		}
		w.WriteHeader(201)

		dec := json.NewDecoder(r.Body)
		m := map[string]interface{}{}
		dec.Decode(&m)
		if m["test"] != "asdf" {
			t.Errorf("Expected payload to be submitted")
		}
		m["id"] = 1337
		enc := json.NewEncoder(w)
		enc.Encode(m)
	})
	defer ts.Close()

	item := map[string]interface{}{
		"test": "asdf",
	}
	err := c.Post("", &item)
	if !apiCalled {
		t.Fatalf("Expected API to be called")
	}
	if err != nil {
		t.Fatalf("Expected no error, got %#v", err)
	}
	if item["id"] != float64(1337) {
		t.Errorf("Expected id to be %#v, got %#v", 1337, item["id"])
	}
}
