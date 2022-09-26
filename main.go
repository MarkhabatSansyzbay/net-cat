package main

import (
	"TCPchat/server"
	"fmt"
	"os"
)

const defaultPort = "8989"

func main() {
	port := defaultPort
	args := len(os.Args[1:])

	if args == 1 {
		port = os.Args[1]
	} else if args != 0 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	server := server.NewServer(port)
	server.HandleConnection()
}
