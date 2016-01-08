package service

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	MutexVar        sync.Mutex
	GlobalTimeOutDB = time.Duration(5000)
	GlobalTimeOutIO = time.Duration(13000)
)

func TimeOutInMilis(duration time.Duration) <-chan time.Time {
	return time.After(duration * time.Millisecond)
}

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

var rtoErrMsg = errors.New("request time out")

func MutexTime() {
	MutexVar.Lock()
	defer MutexVar.Unlock()
}

func ExecuteChannelSqlRow(sequel string) (*sql.Row, error) {
	channelSqlRow := make(chan *sql.Row)
	go QueryRowSQL(sequel, channelSqlRow)
	select {
	case getRow := <-channelSqlRow:
		return getRow, nil
	case <-TimeOutInMilis(GlobalTimeOutDB):
		close(channelSqlRow)
		return nil, rtoErrMsg
	}
}

func ExecuteChannelSqlRows(sequel string) (*sql.Rows, error) {
	chanSqlRows := make(chan QuerySQLType)
	go QuerySQL(sequel, chanSqlRows)
	select {
	case getRows := <-chanSqlRows:
		return getRows.SQLRows, getRows.Error
	case <-TimeOutInMilis(GlobalTimeOutDB):
		close(chanSqlRows)
		return nil, rtoErrMsg
	}
}

func ExecuteInsertSqlResult(sequel string) (int, string, int64) {
	channelSqlResult := make(chan ExecSQLType)
	go ExecSQL(sequel, channelSqlResult)
	select {
	case getResult := <-channelSqlResult:
		err := getResult.Error
		if err != nil {
			status, message := ErrorMessageDB(err.Error())
			return status, message, 0
		} else {
			sqlResult := getResult.SQLResult
			affectedRow, _ := sqlResult.RowsAffected()
			newId, _ := sqlResult.LastInsertId()

			switch {
			case affectedRow < int64(1):
				return 422, "data not efefcted", 0
			default:
				return 200, "success", newId
			}
		}
	case <-TimeOutInMilis(GlobalTimeOutDB):
		close(channelSqlResult)
		return 500, rtoErrMsg.Error(), 0
	}
}

func ExecuteChannelSqlResult(sequel string) (int, string) {
	channelSqlResult := make(chan ExecSQLType)
	go ExecSQL(sequel, channelSqlResult)
	select {
	case getResult := <-channelSqlResult:
		err := getResult.Error
		if err != nil {
			status, message := ErrorMessageDB(err.Error())
			return status, message
		} else {
			sqlResult := getResult.SQLResult
			affectedRow, _ := sqlResult.RowsAffected()
			switch {
			case affectedRow < int64(1):
				return 422, "data not afefcted"
			default:
				return 200, "success"
			}
		}
	case <-TimeOutInMilis(GlobalTimeOutDB):
		close(channelSqlResult)
		return 500, rtoErrMsg.Error()
	}
}

func ExecuteUpdateSqlResult(sequel string) (int, string) {
	status, message := ExecuteChannelSqlResult(sequel)
	if status == 200 {
		return status, "updated"
	} else {
		return status, message
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

func ExecuteCopyFile(out *os.File, file multipart.File, channelCopyFile chan CopyFileType) {
	copied, err := io.Copy(out, file)
	channelCopyFile <- CopyFileType{copied, err}
}
