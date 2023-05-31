package main

import "pictorial/server"

func main() {
	if err := server.New(); err != nil {
		panic(err)
	}
}
