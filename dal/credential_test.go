package dal

import (
	"testing"
)

func TestNewCredential(t *testing.T) {
	c := Credential{}
	t.Logf("%#v", c)
}
