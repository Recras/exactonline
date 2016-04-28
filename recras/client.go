package recras

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/Recras/exactonline/httperror"
)

import (
	log "github.com/Sirupsen/logrus"
)

type basicAuth struct {
	Username string
	Password string
}

type Transport struct {
	Base      http.RoundTripper
	BaseURL   *url.URL
	BasicAuth basicAuth
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

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := cloneRequest(req)

	if t.BaseURL != nil {
		req2.URL.Scheme = t.BaseURL.Scheme
	}
	if req2.URL.Host == "" && t.BaseURL != nil {
		req2.URL.Host = t.BaseURL.Host
	}

	if t.BaseURL != nil && req2.URL.Host == t.BaseURL.Host {
		req2.SetBasicAuth(t.BasicAuth.Username, t.BasicAuth.Password)
	}

	res, err := t.base().RoundTrip(req2)
	return res, err
}

type Client struct {
	Client http.Client
}

func NewClient(hostname, username, password string) Client {
	if hostname[0:7] != "http://" {
		hostname = "https://" + hostname
	}
	b, _ := url.Parse(hostname)
	return Client{http.Client{Transport: &Transport{
		BasicAuth: basicAuth{Username: username, Password: password},
		BaseURL:   b,
	}}}
}

func (c *Client) Get(u string, item interface{}) error {
	log.WithFields(log.Fields{
		"url": u,
	}).Debug("Recras client GET")
	resp, err := c.Client.Get(u)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return httperror.New(resp)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(item)
	return err
}

func (c *Client) Post(u string, item interface{}) error {
	log.WithFields(log.Fields{
		"url":  u,
		"item": item,
	}).Debug("Recras client POST")
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}
	bb := bytes.NewReader(payload)
	resp, err := c.Client.Post(u, "application/json", bb)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return httperror.New(resp)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(item)
	return err
}
