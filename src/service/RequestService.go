// RequestService
package service

import (
	"encoding/json"
	"io"
)

func DecodeJson(v interface{}, body io.ReadCloser) {
	decoder := json.NewDecoder(body)
	decoder.Decode(v)
}
