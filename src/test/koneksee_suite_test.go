package main_test

import (
	"github.com/modocache/gory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"requests"
	"testing"
)

func TestKoneksee(t *testing.T) {
	defineFactories()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Koneksee Suite")
}

func defineFactories() {
	gory.Define("user", requests.User{},
		func(factory gory.Factory) {
			factory["UserName"] = "danilson"
			factory["PhoneNumber"] = "+628973187318"
		})
	gory.Define("userNameEmpty", requests.User{},
		func(factory gory.Factory) {
			factory["UserName"] = ""
			factory["PhoneNumber"] = "+628973187318"
		})
	gory.Define("phoneNumberEmpty", requests.User{},
		func(factory gory.Factory) {
			factory["UserName"] = "jojon"
			factory["PhoneNumber"] = ""
		})
	gory.Define("phoneNumberAndUserEmpty", requests.User{},
		func(factory gory.Factory) {
			factory["UserName"] = ""
			factory["PhoneNumber"] = ""
		})
}
