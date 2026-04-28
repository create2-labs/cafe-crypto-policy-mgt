package main

import (
	"log"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/app"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
