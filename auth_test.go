package exactonline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestExchange(t *testing.T) {
	var called bool

	c := Config{Oauth: &oauth2.Config{}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if code := r.FormValue(`code`); code != `asdf` {
			t.Errorf(`Expected "code" field to be "asdf", got %#v`, code)
		}
		if redirectURL := r.FormValue(`redirect_uri`); redirectURL != c.Oauth.RedirectURL {
			t.Errorf(`Expected "redirect_uri" to be %#v, got %#v`, c.Oauth.RedirectURL, redirectURL)
		}
		if grantType := r.FormValue(`grant_type`); grantType != `authorization_code` {
			t.Errorf(`Expected "grant_type" to be %#v, got %#v`, `authorization_code`, grantType)
		}
		if clientId := r.FormValue(`client_id`); clientId != `asdf` {
			t.Errorf(`Expected "client_id" to be %#v, got %#v`, `asdf`, clientId)
		}
		if clientSecret := r.FormValue(`client_secret`); clientSecret != `s3cr1t` {
			t.Errorf(`Expected "client_secret" to be %#v, got %#v`, `s3cr1t`, clientSecret)
		}
		w.Header().Set(`Content-Type`, `application/json`)
		fmt.Fprintln(w, `{"access_token": "accessToken", "token_type": "bearer", "expires_in": 600, "refresh_token": "refreshToken"}`)
	}))
	defer ts.Close()

	c.Oauth.Endpoint.TokenURL = ts.URL
	_, err := c.Exchange(`asdf`)
	if err != ErrNoClientSecret {
		t.Errorf(`Expected ErrNoClientSecret`)
	}
	c.Oauth.ClientID = `asdf`
	c.Oauth.ClientSecret = `s3cr1t`
	tok, err := c.Exchange(`asdf`)

	if !called {
		t.Error(`Expected mock http server to be called`)
	}
	if err != nil {
		t.Errorf(`Did not expect error, got %#v`, err)
	}
	if tok.AccessToken != `accessToken` {
		t.Errorf(`Expected access token, got %#v`, tok.AccessToken)
	}
	if tok.RefreshToken != `refreshToken` {
		t.Errorf(`Expected refresh token, got %#v`, tok.RefreshToken)
	}
	if tok.TokenType != `bearer` {
		t.Errorf(`Expected bearer token type, got %#v`, tok.TokenType)
	}
	if (tok.Expiry == time.Time{}) {
		t.Errorf(`Expected expiry to be set`)
	}
}

func TestRefreshNoClientSecret(t *testing.T) {
	c := Config{Oauth: &oauth2.Config{}}
	c.Oauth.ClientSecret = ""

	if _, err := c.refreshToken(nil); err != ErrNoClientSecret {
		t.Errorf(`Expected ErrNoClientSecret`)
	}
}

func mockRefreshTokenServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rt := r.FormValue("refresh_token"); rt != "refreshtoken" {
			t.Errorf(`Expected "refresh_token" form value to be "refreshtoken", got %#v`, rt)
		}
		if rt := r.FormValue("client_secret"); rt != "s3cr1t" {
			t.Errorf(`Expected "client_secret" form value to be "s3cr1t", got %#v`, rt)
		}
		if rt := r.FormValue("client_id"); rt != "asdfasdf" {
			t.Errorf(`Expected "client_id" form value to be "asdfasdf", got %#v`, rt)
		}
		if val := r.FormValue("grant_type"); val != "refresh_token" {
			t.Errorf(`Expected "grant_type" form value to be "refresh_token", got %#v`, val)
		}

		w.Header().Set(`Content-Type`, `application/json`)
		fmt.Fprintln(w, `{"access_token": "accessToken", "token_type": "bearer", "expires_in": "600", "refresh_token": "refreshToken"}`)
	}))
}

func TestRefreshWithClientSecret(t *testing.T) {
	ts := mockRefreshTokenServer(t)
	defer ts.Close()

	c := Config{Oauth: &oauth2.Config{}}
	c.Oauth.Endpoint.TokenURL = ts.URL
	c.Oauth.ClientID = "asdfasdf"
	c.Oauth.ClientSecret = "s3cr1t"

	tok, err := c.refreshToken(&oauth2.Token{
		RefreshToken: "refreshtoken",
	})
	if err != nil {
		t.Errorf("Did not expect error, got %#v", err)
	}
	if tok.AccessToken != `accessToken` {
		t.Errorf(`Expected access token, got %#v`, tok.AccessToken)
	}
	if tok.RefreshToken != `refreshToken` {
		t.Errorf(`Expected refresh token, got %#v`, tok.RefreshToken)
	}
	if tok.TokenType != `bearer` {
		t.Errorf(`Expected bearer token type, got %#v`, tok.TokenType)
	}
	if tok.Expiry.Sub(time.Now()) < 590*time.Second {
		t.Errorf(`Expected expiry to be set`)
	}
}

func TestTokenSourceUnexpiredToken(t *testing.T) {
	mock := mockRefreshTokenServer(t)
	defer mock.Close()

	c := Config{Oauth: &oauth2.Config{}}
	c.Oauth.Endpoint.TokenURL = mock.URL

	ts := &tokenSource{
		token: oauth2.Token{
			RefreshToken: "refreshtoken",
			Expiry:       time.Now().Add(3 * time.Second),
		},
		config: c,
	}
	tok, err := ts.Token()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if ts.token != *tok {
		t.Errorf("Expected token to be the same, got %#v and %#v", ts.token, tok)
	}
}

func TestTokenSourceExpiredToken(t *testing.T) {
	mock := mockRefreshTokenServer(t)
	defer mock.Close()

	c := Config{Oauth: &oauth2.Config{}}
	c.Oauth.Endpoint.TokenURL = mock.URL
	c.Oauth.ClientID = "asdfasdf"
	c.Oauth.ClientSecret = "s3cr1t"

	ts := &tokenSource{
		token: oauth2.Token{
			AccessToken:  `accessToken`,
			RefreshToken: `refreshtoken`,
			TokenType:    `bearer`,
			Expiry:       time.Now().Add(-1 * time.Second),
		},
		config: c,
	}
	tok, err := ts.Token()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if tok.Expiry.Before(time.Now()) {
		t.Errorf("Expected expiry to be updated")
	}
	if tok.AccessToken != `accessToken` {
		t.Errorf(`Expected access token, got %#v`, tok.AccessToken)
	}
	if tok.RefreshToken != `refreshToken` {
		t.Errorf(`Expected refresh token, got %#v`, tok.RefreshToken)
	}
	if tok.TokenType != `bearer` {
		t.Errorf(`Expected bearer token type, got %#v`, tok.TokenType)
	}
}

func TestNewTokenSource(t *testing.T) {
	c := Config{Oauth: &oauth2.Config{}}
	tok := c.NewTokenSource("refreshToken")
	if tok.token.RefreshToken != "refreshToken" {
		t.Errorf("Expected RefreshToken value to be `refreshToken`, got %s", tok.token.RefreshToken)
	}
}
