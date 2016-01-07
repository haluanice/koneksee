// Router
package main

import (
	"net/http"
	"os"

	"controller"
	"fmt"
	"responses"
	"service"

	"github.com/drone/routes"
)

func Routes() {
	mux := routes.New()

	mux.Post("/api/v1/users", controller.CreateUser)
	mux.Post("/api/v1/users/token", controller.GenerateNewToken)

	mux.Filter(FilterToken)
	mux.Post("/api/v1/users/index", controller.GetUsers)

	mux.Put("/api/v1/users", controller.UpdateUser)
	mux.Get("/api/v1/users/:id", controller.GetUser)
	mux.Del("/api/v1/users", controller.DeleteUser)
	mux.Put("/api/v1/users/mobile_phone", controller.UpdatePhoneNumber)

	mux.Put("/api/v1/users/file", controller.UploadFile)

	mux.Post("/api/v1/users/action/block", controller.BlockFriend)
	mux.Post("/api/v1/users/action/hide", controller.HideFriend)
	mux.Del("/api/v1/users/action/block", controller.UnBlockFriend)
	mux.Del("/api/v1/users/action/hide", controller.UnHideFriend)

	pwd, _ := os.Getwd()
	mux.Static("/static", pwd)

	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

func FilterToken(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	methodAllowed := method == "POST"

	url := fmt.Sprintf("%s", r.URL)
	listExceptionURL := (url == "/api/v1/users" || url == "/api/v1/users/token")
	service.SetHeaderParameter(w)
	switch {
	case methodAllowed && listExceptionURL:
		return
	default:
		status, message, _, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
		if status != 200 {
			w.WriteHeader(status)
			routes.ServeJson(w, responses.ErrorMessage{status, message})
		}
	}
}
