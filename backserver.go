package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"shareReadWrite/shareReadWrite"
)

func main() {
	rpc.Register(&shareReadWrite.Node{})
	l, err := net.Listen("tcp", ":9876")
	if err != nil {
		log.Fatal("listen error:", err)

	}
	for {
		http.Serve(l, nil)
	}
}
