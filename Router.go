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
	mux.Get("/api/users", controller.GetUsers)
	mux.Get("/api/users/:id", controller.GetUser)
	mux.Post("/api/users", controller.CreateUser)
	mux.Put("/api/users", controller.UpdateUser)
	mux.Del("/api/users", controller.DeleteUser)
	mux.Post("/api/users/token", controller.GenerateNewToken)
	pwd, _ := os.Getwd()
	mux.Static("/static", pwd)

	mux.Post("/api/users/file", controller.UploadFile)
	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}
