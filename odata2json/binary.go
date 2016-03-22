package odata2json

import "encoding/base64"

type Binary []byte

func (b *Binary) MarshalJSON() ([]byte, error) {
	if *b == nil {
		return []byte(`null`), nil
	}
	if len(*b) == 0 {
		return []byte(`""`), nil
	}
	return []byte(`"` + base64.StdEncoding.EncodeToString(*b) + `"`), nil
}
