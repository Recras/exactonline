package exactonline

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"
)

var ErrNoClientSecret = errors.New("Exact auth config has no Client Secret, please set EXACT_CLIENT_SECRET environment")

type Config struct {
	Oauth   *oauth2.Config
	BaseURL string
}

const (
	BASEURL_NL = "https://start.exactonline.nl"
	BASEURL_BE = "https://start.exactonline.be"
	BASEURL_DE = "https://start.exactonline.de"
	BASEURL_UK = "https://start.exactonline.co.uk"
	BASEURL_US = "https://start.exactonline.com"
)

// EnvConfig builds a Config from the EXACT_* environment variables
// If there are environment variables missing, the corresponding values are empty
func EnvConfig() Config {
	baseUrl := os.Getenv("EXACT_BASE_URL")
	if baseUrl == "" {
		baseUrl = BASEURL_NL
	}
	return Config{
		BaseURL: baseUrl,
		Oauth: &oauth2.Config{
			ClientID:     os.Getenv("EXACT_CLIENT_ID"),
			ClientSecret: os.Getenv("EXACT_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("EXACT_REDIRECT_URL"),
			Endpoint: oauth2.Endpoint{
				AuthURL:  baseUrl + "/api/oauth2/auth",
				TokenURL: baseUrl + "/api/oauth2/token",
			},
		},
	}
}

type exactToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in,string"`
	TokenType    string `json:"token_type"`
}

func (c Config) Exchange(code string) (*oauth2.Token, error) {
	if c.Oauth.ClientSecret == "" {
		return nil, ErrNoClientSecret
	}
	data := url.Values{}
	data.Add("code", code)
	data.Add("redirect_uri", c.Oauth.RedirectURL)
	data.Add("grant_type", "authorization_code")
	data.Add("client_id", c.Oauth.ClientID)
	data.Add("client_secret", c.Oauth.ClientSecret)

	resp, err := http.PostForm(c.Oauth.Endpoint.TokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	tok := &oauth2.Token{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(tok)
	if err != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return nil, errors.New("auth.Exchange: could not decode json: " + buf.String())
	}
	tok.Expiry = time.Now().Add(600 * time.Second)
	return tok, nil
}

func (c Config) refreshToken(t *oauth2.Token) (*oauth2.Token, error) {
	if c.Oauth.ClientSecret == "" {
		return nil, ErrNoClientSecret
	}
	data := url.Values{}
	data.Add("refresh_token", t.RefreshToken)
	data.Add("client_secret", c.Oauth.ClientSecret)
	data.Add("client_id", c.Oauth.ClientID)
	data.Add("grant_type", "refresh_token")
	resp, err := http.PostForm(c.Oauth.Endpoint.TokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	et := &exactToken{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(et)
	if err != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return nil, errors.New("auth.refreshToken: could not decode json: " + buf.String())
	}
	return &oauth2.Token{
		AccessToken:  et.AccessToken,
		RefreshToken: et.RefreshToken,
		TokenType:    et.TokenType,
		Expiry:       time.Now().Add(time.Duration(et.ExpiresIn) * time.Second),
	}, nil
}

type tokenSource struct {
	token  oauth2.Token
	config Config
}

func (ts tokenSource) Token() (*oauth2.Token, error) {
	if time.Now().Before(ts.token.Expiry) {
		return &ts.token, nil
	}
	tok, err := ts.config.refreshToken(&ts.token)
	return tok, err
}

func (c Config) NewTokenSource(refresh_token string) tokenSource {
	tok := tokenSource{
		config: c,
	}
	tok.token.RefreshToken = refresh_token
	return tok
}
