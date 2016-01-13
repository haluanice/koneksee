package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/drone/routes"
	_ "github.com/go-martini/martini"
	"github.com/modocache/gory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"model"
	"net/http"
	"net/http/httptest"
	"service"
)

func slice(data interface{}) map[string]interface{} {
	mapData, _ := data.(map[string]interface{})
	return mapData
}

var _ = Describe("Server", func() {
	var server *routes.RouteMux
	var request *http.Request
	var recorder *httptest.ResponseRecorder
	var newId float64
	var token string
	BeforeEach(func() {
		server = Routes("koneksee_test")
		recorder = httptest.NewRecorder()
	})

	Describe("POST /api/v1/users", func() {
		Context("with username and phone_number", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("user"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "281982918291")
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(201))
			})
		})
		Context("with the same phone_number", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("userOverridePhone"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "281982918291")
			})
			It("returns a status code of 200", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(200))
			})
		})
		Context("username empty", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("userNameEmpty"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "281982918291")
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)
				user := model.GeneralMsg{}
				bod := recorder.Body
				buff, _ := ioutil.ReadAll(bod)
				err := json.Unmarshal(buff, &user)
				if err != nil {
					return
				}
				mapData := slice(user.Data)
				newId, _ = mapData["user_id"].(float64)
				token, _ = mapData["token"].(string)
				Expect(recorder.Code).To(Equal(201))
			})
		})
		Context("phone number empty", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("phoneNumberEmpty"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "281982918291")
			})
			It("phone number empty returns 422", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(422))
			})
		})
		Context("username and phone number empty", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("phoneNumberAndUserEmpty"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "281982918291")
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(422))
			})
		})
	})
	Describe("PUT /api/v1/users/:id/user_name", func() {
		Context("update user name", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("updateUserName"))
				url := fmt.Sprintf("/api/v1/users/%v/user_name", newId)
				request, _ = http.NewRequest(
					"PUT", url, bytes.NewReader(body))
				tokenAuth := fmt.Sprintf("Bearer %v", token)
				request.Header.Set("Authorization", tokenAuth)
			})
			It("update username returns 200", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})

	})
	Describe("PUT /api/v1/users/:id/avatar", func() {
		// TO DO
	})
	Describe(fmt.Sprintf("GET /api/v1/users/1"), func() {
		Context("view selected person", func() {
			BeforeEach(func() {
				url := fmt.Sprintf("/api/v1/users/%v", newId)
				request, _ = http.NewRequest("GET", url, nil)
				tokenAuth := fmt.Sprintf("Bearer %v", token)
				request.Header.Set("Authorization", tokenAuth)
			})
			It("returns a status code of 200", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(200))
			})
		})
	})
	Describe("POST /api/v1/users/sync", func() {
		Context("view index person", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("contact"))
				request, _ = http.NewRequest("POST", "/api/v1/users/sync", bytes.NewReader(body))
				tokenAuth := fmt.Sprintf("Bearer %v", token)
				request.Header.Set("Authorization", tokenAuth)
			})
			AfterEach(func() {
				chanDelete := make(chan service.ExecSQLType)
				go service.ExecSQL("truncate table users", chanDelete)
				_ = <-chanDelete
			})
			It("returns a status code of 200", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(200))
			})
		})
	})
})
