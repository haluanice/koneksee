package service

import (
	"bytes"

	"github.com/kyokomi/cloudinary"
	"golang.org/x/net/context"
)

type CloudynaryInfo struct {
	FilePath string
	Err      error
}

var (
	CtxCloudinary  = NewCloudinary()
	CloudinaryAuth = "cloudinary://829347955498358:M1YyDwa7BSdNHS4qqeUQW3l6S4A@dxclwskoq"
)

func NewCloudinary() context.Context {
	ctx := context.Background()
	ctxCloud := cloudinary.NewContext(ctx, CloudinaryAuth)
	return ctxCloud
}

func UploadImage(nameFile string, buff []byte) chan CloudynaryInfo {
	readFileCopied := bytes.NewReader(buff)
	chanInfo := make(chan CloudynaryInfo)
	go func() {
		err := cloudinary.UploadStaticImage(CtxCloudinary, nameFile, readFileCopied)
		chanInfo <- CloudynaryInfo{cloudinary.ResourceURL(CtxCloudinary, nameFile), err}
	}()
	return chanInfo
}
