package main_test

import (
	"github.com/modocache/gory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"model"
	"testing"
)

func TestKoneksee(t *testing.T) {
	defineFactories()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Koneksee Suite")
}

func defineFactories() {
	gory.Define("user", model.User{},
		func(factory gory.Factory) {
			factory["UserName"] = "danilson"
			factory["PhoneNumber"] = "+628973187318"
		})
	gory.Define("userNameEmpty", model.User{},
		func(factory gory.Factory) {
			factory["UserName"] = ""
			factory["PhoneNumber"] = "+621973187318"
		})
	gory.Define("userOverridePhone", model.User{},
		func(factory gory.Factory) {
			factory["UserName"] = "joko"
			factory["PhoneNumber"] = "+628973187318"
		})
	gory.Define("phoneNumberEmpty", model.User{},
		func(factory gory.Factory) {
			factory["UserName"] = "jojon"
			factory["PhoneNumber"] = ""
		})
	gory.Define("phoneNumberAndUserEmpty", model.User{},
		func(factory gory.Factory) {
			factory["UserName"] = ""
			factory["PhoneNumber"] = ""
		})
	gory.Define("updateUserName", model.User{},
		func(factory gory.Factory) {
			factory["UserName"] = "messi"
		})
	gory.Define("contact", model.ContactList{},
		func(factory gory.Factory) {
			factory["Contact"] = []string{"+621973187318", "+28198291829"}
		})
}
