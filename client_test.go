package exactonline

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestNewClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if acc := r.Header.Get("Accept"); acc != "application/json" {
			t.Errorf("Expected Accept header to be application/json, got %#v", acc)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer opensesame" {
			t.Errorf("Expected request to have valid credentials, got %#v", auth)
		}
	}))
	defer ts.Close()

	c := Config{}
	cl := c.NewClient(oauth2.Token{
		AccessToken: "opensesame",
		Expiry:      time.Now().Add(2 * time.Second),
		TokenType:   "bearer",
	})
	cl.Client.Get(ts.URL)
}

func TestRoundTrip(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if acc := r.Header.Get("Accept"); acc != "application/json" {
			t.Errorf("Expected Accept header to be application/json, got %s", acc)
		}

		if r.Method == "POST" {
			if p := r.Header.Get("Prefer"); p != "return=representation" {
				t.Errorf("Expected Prefer header to be return=representation, got %s", p)
			}
		}
	}))
	defer ts.Close()
	c := &http.Client{Transport: &Transport{}}
	c.Get(ts.URL)
	c.Post(ts.URL, "", nil)
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
