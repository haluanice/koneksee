package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/modocache/gory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
	"service"
)

var _ = Describe("Server", func() {
	var server *martini.ClassicMartini
	var request *http.Request
	var recorder *httptest.ResponseRecorder
	BeforeEach(func() {
		server = Routes()
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
				Expect(recorder.Code).To(Equal(201))
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
})
