package recras

import (
	"errors"
	"fmt"
	"net/http"
)

var ErrInvalidHostname = errors.New("Invalid Recras hostname")
var ErrInvalidCredentials = errors.New("Invalid Recras credentials")

func IsValidUser(recrasHostname, user, password string) error {
	if !isValidHostname(recrasHostname) {
		return ErrInvalidHostname
	}
	recrasURL := fmt.Sprintf("https://%s", recrasHostname)
	return isValidUser(recrasURL, user, password)
}
func isValidUser(recrasURL, user, password string) error {
	r, err := http.NewRequest("GET", recrasURL+"/api2/personeel/me", nil)
	if err != nil {
		return err
	}
	r.SetBasicAuth(user, password)
	c := &http.Client{}
	resp, err := c.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return ErrInvalidCredentials
	}
	return nil
}

func isValidHostname(hostname string) bool {
	return len(hostname) > 10 && hostname[len(hostname)-10:] == ".recras.nl"
}
