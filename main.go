package main

import (
	"log"

	"github.com/stratumfarm/phantomias/cmd"
	_ "github.com/stratumfarm/phantomias/docs"
)

// @title StratumFarm Pool API
// @version 1.0
// @description This is the public pool api from stratum.farm
// @termsOfService https://stratum.farm/terms/
// @contact.name StratumFarm Support
// @contact.email pool@stratum.farm
// @host 152.228.229.130:3000
// @BasePath /

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
