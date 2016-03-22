package recras

import (
	"errors"
	"fmt"
	"net/http"
)

var ErrInvalidCredentials = errors.New("Invalid Recras credentials")

func IsValidUser(recrasHostname, user, password string) error {
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
	if resp.StatusCode == 401 {
		return ErrInvalidCredentials
	}
	return nil
}
