package main

import (
	"pictorial/log"
	"pictorial/server"
)

func main() {
	log.Logger.Info("welcome!")
	if err := server.New(); err != nil {
		panic(err)
	}
}
