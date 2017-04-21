package httperror

import (
	"fmt"
	"net/http"
	"io/ioutil"
)

type HTTPError struct {
	StatusCode int
	Request    *http.Request
	Response   *http.Response
}

func (e HTTPError) Error() string {
	defer e.Response.Body.Close()
	body, err := ioutil.ReadAll(e.Response.Body)
	if err != nil {
		return "ioutil.ReadAll failed"
	}
	return fmt.Sprintf("HTTP error %d for %s %s: %s", e.StatusCode, e.Request.Method, e.Request.URL, body)
}

func New(resp *http.Response) HTTPError {
	return HTTPError{
		StatusCode: resp.StatusCode,
		Request:    resp.Request,
		Response:   resp,
	}
}
