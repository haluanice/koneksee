package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/modocache/gory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"responses"
	"service"
)

var _ = Describe("Server", func() {
	var server *martini.ClassicMartini
	var request *http.Request
	var recorder *httptest.ResponseRecorder
	var newId int
	var token string
	BeforeEach(func() {
		server = Routes("koneksee_test")
		recorder = httptest.NewRecorder()
	})
	AfterEach(func() {
		chanDelete := make(chan service.ExecSQLType)
		go service.ExecSQL("truncate table users", chanDelete)
		_ = <-chanDelete
	})
	Describe("POST /api/v1/users", func() {
		Context("with username and phone_number", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("user"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "1234")
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)

				user := responses.GeneralMsg{}
				Expect(recorder.Code).To(Equal(201))
				bod := recorder.Body
				buff, _ := ioutil.ReadAll(bod)
				err := json.Unmarshal(buff, &user)
				if err != nil {
					return
				}
				mapData, _ := user.Data.(map[string]interface{})
				tempId, _ := mapData["user_id"].(float64)
				token, _ = mapData["token"].(string)
				newId = int(tempId)
				fmt.Println("New ID", newId)
			})
		})

		Context("username empty", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("userNameEmpty"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "1234")
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(201))
			})
		})
		Context("phone number empty", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("phoneNumberEmpty"))
				request, _ = http.NewRequest(
					"POST", "/api/v1/users", bytes.NewReader(body))
				request.Header.Set("Authorization", "1234")
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
				request.Header.Set("Authorization", "1234")
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(422))
			})
		})
	})
	Describe("PUT /api/v1/users/:id/user_name", func() {

	})
	Describe("PUT /api/v1/users/:id/avatar", func() {

	})
	Describe(fmt.Sprintf("GET /api/v1/users/%s", newId), func() {
		Context("view selected person", func() {
			// url := fmt.Sprintf("/api/v1/users/%v", 1)
			// BeforeEach(func() {
			// 	request, _ = http.NewRequest("GET", url, nil)
			// 	tokenAuth := fmt.Sprintf("Bearer %v", token)
			// 	request.Header.Set("Authorization", tokenAuth)
			// })
			// It("returns a status code of 200", func() {
			// 	server.ServeHTTP(recorder, request)
			// 	fmt.Println(token)
			// 	fmt.Println(url)
			// 	Expect(recorder.Code).To(Equal(200))
			// })
		})

	})
	Describe("POST /api/v1/users/index", func() {

	})
})
