package main_test

import (
	"bytes"
	"encoding/json"
	"github.com/drone/routes"
	"github.com/modocache/gory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Server", func() {
	var server Server
	var request *http.Request
	var recorder *httptest.ResponseRecorder
	BeforeEach(func() {
		server = routes.New()
		recorder = httptest.NewRecorder()
	})
	AfterEach(func() {

	})
	Describe("POST /api/v1/users", func() {
		Context("with username and phone_number", func() {
			BeforeEach(func() {
				body, _ := json.Marshal(
					gory.Build("user"))
				request, _ = http.NewRequest(
					"POST", "localhost:8080/api/v1/users", bytes.NewReader(body))
			})
			It("returns a status code of 201", func() {
				server.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(200))
			})
		})
	})
})
