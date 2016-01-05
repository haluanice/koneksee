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
var ChannelCopyFile = make(chan CopyFileType)

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

func ExecuteCopyFile(out *os.File, file multipart.File) {
	copied, err := io.Copy(out, file)
	ChannelCopyFile <- CopyFileType{copied, err}
}

func ExecuteChanelSqlRows(sequel string) (*sql.Rows, error) {
	go QuerySQL(sequel, ChanelSqlRows)
	getRows := <-ChanelSqlRows
	return getRows.SQLRows, getRows.Error
}

func ExecuteChanelSqlResult(sequel string) (sql.Result, error) {
	go ExecSQL(sequel, ChanelSqlResult)
	getResult := <-ChanelSqlResult
	return getResult.SQLResult, getResult.Error
}

func GenerateNewPath(path string, fileType string) (pathFile string, nameFile string, err error) {
	uuid, err := exec.Command("uuidgen").Output()
	//t := time.Now().Format(time.RFC850)
	nameFile = fmt.Sprintf("%x.%s", uuid, fileType)
	pathFile = fmt.Sprintf("%s%s", path, nameFile)
	return
}

func CreateFile(pathFile string) (output *os.File, err error) {
	output, err = os.Create(pathFile)
	return
}
