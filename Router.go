// Router
package main

import (
	"net/http"

	"controller"
	"fmt"
	"github.com/drone/routes"
	"model"
	"service"
	"strconv"
	"strings"
)

func Run(dbName string) {
	mux := Routes(dbName)
	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
	// mux.Run()
}
func Routes(dbName string) *routes.RouteMux {
	service.NewDatabase(dbName)
	mux := routes.New()
	UnAuthorizedGroup(mux)
	mux.Filter(FilterToken)
	AutHorizedGroup(mux)
	return mux
}

func UnAuthorizedGroup(mux *routes.RouteMux) {
	mux.Post("/api/v1/users", controller.CreateUser)
	mux.Post("/api/v1/users/token", controller.GenerateNewToken)
	//mux.Static("/static", service.GetRootPath())
}

func AutHorizedGroup(mux *routes.RouteMux) {
	mux.Post("/api/v1/users/sync", controller.GetUsers)
	mux.Get("/api/v1/users/:id/blocked", controller.GetUsersBlocked)

	mux.Put("/api/v1/users/:id/user_name", controller.UpdateUserName)
	mux.Put("/api/v1/users/:id/status", controller.UpdateUserStatus)
	mux.Get("/api/v1/users/:id", controller.GetUser)
	mux.Del("/api/v1/users/:id", controller.DeleteUser)
	mux.Put("/api/v1/users/:id/mobile_phone", controller.UpdatePhoneNumber)
	mux.Put("/api/v1/users/:id", controller.UpdateUserProfile)

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
		if r.Header.Get(("Authorization")) != "281982918291" {
			w.WriteHeader(http.StatusUnauthorized)
			routes.ServeJson(w, model.DefaultMessage{http.StatusUnauthorized, "anauthorized"})
		}
		return
	//TO DO: case create user auth header for api_key & secret_api
	default:
		id := r.URL.Query().Get(":id")
		status, message, mobilePhone := service.GetTokenHeader(r.Header.Get("Authorization"))
		r.Header.Set("phone_number", mobilePhone)
		r.Header.Set("user_id", id)
		r.Header.Set("status_filter", strconv.Itoa(status))
		if status != 200 {
			w.WriteHeader(status)
			routes.ServeJson(w, model.DefaultMessage{status, message})
		}
	}
}
