package main

import (
	"github.com/thejackrabbit/email/server"
)

func main() {

	serv := server.NewEmailServer()
	serv.Run()

}
