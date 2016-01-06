package service

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
)

var ChanelSqlRows = make(chan QuerySQLType)
var ChanelSqlRow = make(chan *sql.Row)
var ChanelSqlResult = make(chan ExecSQLType)
var ChanelCopyFile = make(chan CopyFileType)

type ExecSQLType struct {
	SQLResult sql.Result
	Error     error
}

type QuerySQLType struct {
	SQLRows *sql.Rows
	Error   error
}

type CopyFileType struct {
	Copy int64
	Err  error
}

func ExecuteChanelSqlRow(sequel string) *sql.Row {
	go QueryRowSQL(sequel, ChanelSqlRow)
	getRow := <-ChanelSqlRow
	return getRow
}

func ExecuteChanelSqlRows(sequel string) (*sql.Rows, error) {
	go QuerySQL(sequel, ChanelSqlRows)
	getRows := <-ChanelSqlRows
	return getRows.SQLRows, getRows.Error
}

func ExecuteInsertSqlResult(sequel string) (int, string, int64) {
	go ExecSQL(sequel, ChanelSqlResult)
	getResult := <-ChanelSqlResult
	err := getResult.Error
	if err != nil{
		status, message := ErrorMessageDB(err.Error())
		return status, message, 0
	}else{
		sqlResult := getResult.SQLResult
		affectedRow, _ := sqlResult.RowsAffected()
		newId, _ := sqlResult.LastInsertId()
		
		switch{
		case affectedRow < int64(1):
			return 422, "data not efefcted", 0
		default:
			return 200, "success", newId
		} 
	}
}

func ExecuteChanelSqlResult(sequel string) (int, string) {
	go ExecSQL(sequel, ChanelSqlResult)
	getResult := <-ChanelSqlResult
	err := getResult.Error
	if err != nil{
		status, message := ErrorMessageDB(err.Error())
		return status, message
	}else{
		sqlResult := getResult.SQLResult
		affectedRow, _ := sqlResult.RowsAffected()		
		switch{
		case affectedRow < int64(1):
			return 422, "data not afefcted"
		default:
			return 200, "success"
		} 
	}
}

func ExecuteUpdateSqlResult(sequel string) (int, string){
	status, _ := ExecuteChanelSqlResult(sequel)
	if status == 404 {
		return status, "data not updated"
	}else{
		return status, "data updated"
	}
}

func GenerateNewPath(path string, fileType string) (pathFile string, nameFile string, err error) {
	uuid, err := exec.Command("uuidgen").Output()
	nameFile = fmt.Sprintf("%x.%s", uuid, fileType)
	pathFile = fmt.Sprintf("%s%s", path, nameFile)
	return
}

func CreateFile(pathFile string) (output *os.File, err error) {
	output, err = os.Create(pathFile)
	return
}

func ExecuteCopyFile(out *os.File, file multipart.File) chan CopyFileType {
	copied, err := io.Copy(out, file)
	ChanelCopyFile <- CopyFileType{copied, err}
	return ChanelCopyFile
}
