package service

import (
	"github.com/kyokomi/cloudinary"
	"golang.org/x/net/context"
)

var (
	CtxCloudinary  = NewCloudinary()
	CloudinaryAuth = "cloudinary://829347955498358:M1YyDwa7BSdNHS4qqeUQW3l6S4A@dxclwskoq"
)

func NewCloudinary() context.Context {
	ctx := context.Background()
	ctxCloud := cloudinary.NewContext(ctx, CloudinaryAuth)
	return ctxCloud
}
