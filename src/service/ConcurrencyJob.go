package service

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"time"
	"errors"
	"sync"
)


var MutexVar sync.Mutex
func TimeOutInMilis(duration time.Duration) <-chan time.Time{
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

func MutexTime(){
	MutexVar.Lock()
	defer MutexVar.Unlock()
}

func ExecuteChannelSqlRow(sequel string) (*sql.Row, error) {
	channelSqlRow := make(chan *sql.Row)
	go QueryRowSQL(sequel, channelSqlRow)
	select{
	case getRow := <-channelSqlRow:
		return getRow, nil
	case <-TimeOutInMilis(500):
		return nil, rtoErrMsg
	}
}

func ExecuteChannelSqlRows(sequel string) (*sql.Rows, error) {
	chanSqlRows := make(chan QuerySQLType)
	go QuerySQL(sequel, chanSqlRows)
	
	select{
	case getRows := <-chanSqlRows:
		return getRows.SQLRows, getRows.Error
	case <-TimeOutInMilis(500):
		return nil, rtoErrMsg
	}
}

func ExecuteInsertSqlResult(sequel string) (int, string, int64) {
	channelSqlResult := make(chan ExecSQLType)
	go ExecSQL(sequel, channelSqlResult)
	select{
	case getResult := <-channelSqlResult:
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
	case <-TimeOutInMilis(500):
		return 500, rtoErrMsg.Error(),0
	}
}

func ExecuteChannelSqlResult(sequel string) (int, string) {
	channelSqlResult := make(chan ExecSQLType)
	go ExecSQL(sequel, channelSqlResult)
	select{
	case getResult := <-channelSqlResult:
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
	case <-TimeOutInMilis(500):
		return 500, rtoErrMsg.Error()
	}
}

func ExecuteUpdateSqlResult(sequel string) (int, string){
	status, _ := ExecuteChannelSqlResult(sequel)
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

func ExecuteCopyFile(out *os.File, file multipart.File, channelCopyFile chan CopyFileType){
	copied, err := io.Copy(out, file)
	channelCopyFile <- CopyFileType{copied, err}
}
