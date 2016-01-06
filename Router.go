// Router
package main

import (
	"net/http"
	"os"

	"controller"

	"github.com/drone/routes"
)

func Routes() {
	mux := routes.New()

	mux.Get("/api/v1/users", controller.GetUsers)
	mux.Get("/api/v1/users/:id", controller.GetUser)
	mux.Post("/api/v1/users", controller.CreateUser)
	mux.Put("/api/v1/users", controller.UpdateUser)
	mux.Put("/api/v1/users/file", controller.UploadFile)
	mux.Del("/api/v1/users", controller.DeleteUser)
	mux.Post("/api/v1/users/token", controller.GenerateNewToken)
	mux.Post("/api/v1/users/action/block", controller.BlockFriend)
	mux.Post("/api/v1/users/action/hide", controller.HideFriend)
	mux.Del("/api/v1/users/action/block", controller.UnBlockFriend)
	mux.Del("/api/v1/users/action/hide", controller.UnHideFriend)

	pwd, _ := os.Getwd()
	mux.Static("/static", pwd)
	
	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}
