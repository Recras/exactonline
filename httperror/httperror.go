package httperror

import (
	"fmt"
	"net/http"
)

type HTTPError struct {
	StatusCode int
	Request    *http.Request
	Response   *http.Response
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d for %s %s: %s", e.StatusCode, e.Request.Method, e.Request.URL, e.Response.Body)
}

func New(resp *http.Response) HTTPError {
	return HTTPError{
		StatusCode: resp.StatusCode,
		Request:    resp.Request,
		Response:   resp,
	}
}
