package main

import (
	"Membership/daemon"
	"SDFS/master"
	"log"
	"time"
)

func main() {
	Daemon, err := daemon.NewDaemon()
	if err != nil {
		log.Println("Can not create new daemon on port")
		return
	}
	// only run on Server 1,2,3
	if Daemon.ID == 1 || Daemon.ID == 2 || Daemon.ID == 3 {
		go Daemon.RunMaster()
	}

	if Daemon.ID == 1 {
		Introducer := daemon.NewIntroducer()
		go Introducer.ContactProc()
	}

	// run on every Server
	go Daemon.HandleStdIn()
	go Daemon.UDPListener()

	// Disseminate every 200ms
	// originally 100ms
	for Daemon.Alive {
		Daemon.UpdateAndDisseminate()
		time.Sleep(time.Millisecond * 100)
	}
}
