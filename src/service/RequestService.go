// RequestService
package service

import (
	"encoding/json"
	"fmt"
	"io"
)

func SelectQuery(field string, table string, condition string) string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s", field, table, condition)
}
func UpdateQuery(table string, field string, condition string) (int, string) {
	sequel := fmt.Sprintf("UPDATE %s SET %s WHERE %s ", table, field, condition)
	return ExecuteUpdateSqlResult(sequel)
}

func DecodeJson(v interface{}, body io.ReadCloser) {
	decoder := json.NewDecoder(body)
	decoder.Decode(v)
}
