package exactonline

import (
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

type Client struct {
	Client   http.Client
	Division int
}

var ErrNoDivision = errors.New("exactonline/api: Client has no Division, use Client.GetDefaultDivision first")

// NewClient creates a new Exact Online API client
func (c Config) NewClient(tok oauth2.Token) *Client {
	ts := tokenSource{
		token:  tok,
		config: c,
	}
	_ = ts
	b, _ := url.Parse(c.BaseURL)
	return &Client{
		Client: http.Client{
			Transport: &oauth2.Transport{
				Base:   &Transport{BaseURL: b},
				Source: ts,
			},
		},
	}
}

type Transport struct {
	Base    http.RoundTripper
	BaseURL *url.URL
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := cloneRequest(req)
	req2.Header.Add("Accept", "application/json")

	if req2.URL.Scheme == "" && t.BaseURL != nil {
		req2.URL.Scheme = t.BaseURL.Scheme
	}
	if req2.URL.Host == "" && t.BaseURL != nil {
		req2.URL.Host = t.BaseURL.Host
	}

	if req2.Method == "POST" {
		req2.Header.Add("Prefer", "return=representation")
	}

	res, err := t.base().RoundTrip(req2)
	return res, err
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}
