// Router
package main

import (
	"net/http"

	"controller"
	"fmt"
	"responses"
	"service"
	"strconv"
	"strings"

	"github.com/drone/routes"
)

func Routes() {
	mux := routes.New()
	GroupUnaUthorize(mux)
	GroupAuthorize(mux)
	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

func GroupUnaUthorize(mux *routes.RouteMux) {
	mux.Post("/api/v1/users", controller.CreateUser)
	mux.Post("/api/v1/users/token", controller.GenerateNewToken)
	mux.Static("/static", service.GetRootPath())
}
func GroupAuthorize(mux *routes.RouteMux) {
	mux.Filter(FilterToken)
	mux.Post("/api/v1/users/index", controller.GetUsers)
	mux.Get("/api/v1/users/:id/blocked", controller.GetUsersBlocked)

	mux.Put("/api/v1/users/:id/user_name", controller.UpdateUserName)
	mux.Get("/api/v1/users/:id", controller.GetUser)
	mux.Del("/api/v1/users/:id", controller.DeleteUser)
	mux.Put("/api/v1/users/:id/mobile_phone", controller.UpdatePhoneNumber)

	mux.Put("/api/v1/users/:id/avatar", controller.UploadFile)

	mux.Post("/api/v1/users/:id/block", controller.BlockFriend)
	mux.Del("/api/v1/users/:id/block", controller.UnBlockFriend)
}

func FilterToken(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	url := fmt.Sprintf("%s", r.URL)

	allowedMethodUnAuth := (method == "POST")
	listExceptionURL := (url == "/api/v1/users" || url == "/api/v1/users/token")
	serveStaticPath := (strings.Contains(url, "/static/") && method == "GET")

	if !serveStaticPath {
		service.SetHeaderParameterJson(w)
	}

	switch {
	case serveStaticPath:
		return
	case listExceptionURL && allowedMethodUnAuth:
		return
	//TO DO: case create user auth header for api_key & secret_api
	default:
		status, message, mobilePhone := service.GetTokenHeader(r.Header.Get("Authorization"))
		r.Header.Set("mobile_phone", mobilePhone)
		r.Header.Set("status_filter", strconv.Itoa(status))
		if status != 200 {
			w.WriteHeader(status)
			routes.ServeJson(w, responses.DefaultMessage{status, message})
		}
	}
}
