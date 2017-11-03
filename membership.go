package main

import (
	"Membership/daemon"
	"log"
	"time"
)

// Caller function for Membership protocol
func main() {
	// creat daemon on specific port 4000
	Daemon, err := daemon.NewDaemon()
	if err != nil {
		log.Println("Can not create new daemon on port")
		return
	}
	// Create extra thread on node 1 as Introducer
	if Daemon.ID == 1 {
		Introducer := daemon.NewIntroducer()
		go Introducer.ContactProc()
	}

	go Daemon.HandleStdIn()
	go Daemon.UDPListener()

	// Disseminate every 200ms
	// originally 100ms
	for Daemon.Alive {
		Daemon.UpdateAndDisseminate()
		time.Sleep(time.Millisecond * 100)
	}
}
