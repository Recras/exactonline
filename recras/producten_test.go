package recras

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetAllProducten(t *testing.T) {
	ts, c := createTestAPI(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id": 1, "leverancier_id": 2, "naam": "asdf"}]`)
	})
	defer ts.Close()
	_ = c
}
