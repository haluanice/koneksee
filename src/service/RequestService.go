// RequestService
package service

import (
	"encoding/json"
	"fmt"
	"io"
)

func UpdateQuery(table string, field string, condition string) string {
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s ", table, field, condition)
}

func DecodeJson(v interface{}, body io.ReadCloser) {
	decoder := json.NewDecoder(body)
	decoder.Decode(v)
}
