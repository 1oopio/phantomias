package main

import (
	"log"

	"github.com/1oopio/phantomias/cmd"
	_ "github.com/1oopio/phantomias/docs" // swagger docs
)

// @title 1oop Pool API
// @version 1.0
// @description This is the public pool api from 1oop.io
// @termsOfService https://1oop.io/terms/
// @contact.name 1oop Support
// @contact.email pool@1oop.io
// @host 152.228.229.130:3000
// @BasePath /

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
