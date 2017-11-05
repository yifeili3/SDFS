package main

import (
	"SDFS/daemon"
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
		// run master
		Master := master.NewMaster()
		go Master.UDPListener()
		go Master.DisseminateMeta()
	}

	if Daemon.ID == 1 {
		Introducer := daemon.NewIntroducer()
		go Introducer.ContactProc()
	}

	// run on every Server
	go Daemon.HandleStdIn()
	go Daemon.UDPListener()
	go Daemon.SDFSListener()

	// Disseminate every 200ms
	// originally 100ms
	for Daemon.Alive {
		Daemon.UpdateAndDisseminate()
		time.Sleep(time.Millisecond * 100)
	}
}
