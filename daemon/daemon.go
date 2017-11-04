package daemon

import (
	"Membership/member"
	"Membership/util"
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
<<<<<<< HEAD
	"os/exec"
=======
	"os"
>>>>>>> 3780bff5f8eef76fbd61f74b488e88263dc68c80
	"strconv"
	"strings"
)

//metadata: filename, []ReplicaList, timestamp, filesize

const (
	serverBase      = "172.22.154.132"
	threshold       = 20
	contactID       = 1
	contactListener = 4096
	contactSender   = 4002
	udpport         = 4000
	sdfsport        = 4004
	sdfsDir         = "/home/yifeili3/sdfs/"
	localDir        = "/home/yifeili3/local/"
)

// Daemon process
type Daemon struct {
	Connection     *net.UDPConn
	Addr           net.UDPAddr
	ID             int
	MembershipList []member.Node
	MetaData       map[string]MetaInfo
	SendList       []int
	MonitorList    []int
	Alive          bool
	Active         bool
	SDFSConnection *net.UDPConn
	SDFSUDPAddr    net.UDPAddr
	Replica        chan []int
}

// Introducer process
type Introducer struct {
	MembershipList []member.Node
	Addr           net.UDPAddr
}

type Message struct {
	Cmd          string
	SdfsFileName string
}

type RPCMeta struct {
	ReplicaList []int
	Command     Message
}

type MetaInfo struct {
	Filename    string
	ReplicaList []int
	Timestamp   int
	FileSize    int
}

// Construct newIntroducer
func NewIntroducer() (in *Introducer) {
	addr := net.UDPAddr{
		Port: contactListener,
		IP:   net.ParseIP(serverBase),
	}
	in = &Introducer{
		Addr:           addr,
		MembershipList: make([]member.Node, 10),
	}
	for i := 0; i < 10; i++ {
		in.MembershipList[i] = *member.NewMember(i+1, net.UDPAddr{IP: net.ParseIP(calculateIP(i + 1)), Port: udpport}, 0)
	}
	return in
}

// Construct newDaemon ...
func NewDaemon() (daemon *Daemon, err error) {
	serverID := whoAmI()
	ipAddr := whereAmI()
	log.Println("Create daemon process on node ", serverID)
	util.WriteLog(serverID, "Create daemon process on node "+strconv.Itoa(serverID))

	/*   Initialize membership     */
	addr := net.UDPAddr{
		Port: udpport,
		IP:   net.ParseIP(ipAddr),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Println("Can not create UDP listener: ", err)
		util.WriteLog(serverID, "Can not create UDP listener.")
	}

	sdfsaddr := net.UDPAddr{
		Port: sdfsport,
		IP:   net.ParseIP(ipAddr),
	}
	sdfsconn, err := net.ListenUDP("udp", &sdfsaddr)
	if err != nil {
		log.Println("Can not create SDFSUDP listener: ", err)
		util.WriteLog(serverID, "Can not create UDP listener.")
	}

	daemon = &Daemon{
		Connection:     conn,
		Addr:           addr,
		ID:             serverID,
		MembershipList: make([]member.Node, 10),
		Alive:          true,
		Active:         false,
		SDFSConnection: sdfsconn,
		SDFSUDPAddr:    sdfsaddr,
	}
	//fill member in memberlist
	for i := 0; i < 10; i++ {
		daemon.MembershipList[i] = *member.NewMember(i+1, net.UDPAddr{IP: net.ParseIP(calculateIP(i + 1)), Port: udpport}, 0)
	}

	/*   Initialize SDFS     */
	daemon.clearSDFS()

	return daemon, err
}

// HandleStdIn ...
// Thread to wait for standard input
func (d *Daemon) HandleStdIn() {
	var input string

	inputReader := bufio.NewReader(os.Stdin)
	// JOIN LEAVE LIST LISTID
	for {
		fmt.Scanf("%q", &input)
		fmt.Println("Enter the operation:")
		input, _ = inputReader.ReadString('\n')
		command := strings.Split(input, " ")
		if len(command) == 1 {
			if command[0] == "JOIN" {
				d.joinGroup()
			} else if command[0] == "LEAVE" {
				log.Println("node " + strconv.Itoa(d.ID) + " leaves...")
				util.WriteLog(d.ID, "node "+strconv.Itoa(d.ID)+" leaves...")
				d.leaveGroup()
			} else if command[0] == "LIST" {
				d.listGroup()
			} else if command[0] == "LISTID" {
				log.Println("Current ID: " + strconv.Itoa(d.ID))
			} else if command[0] == "YES" {
				// handle second write
			} else if command[0] == "NO" {
				// reject second write
			} else if command[0] == "STORE" {
				d.store()
			} else {
				log.Println("Please enter valid command!")
				continue
			}
		} else {
			if len(command) == 3 && command[0] == "PUT" {
				d.put(command[1], command[2])
			} else if len(command) == 3 && command[0] == "GET" {
				d.get(command[1], command[2])
			} else if len(command) == 2 && command[0] == "DELETE" {
				d.delete(command[1])
			} else if len(command) == 2 && command[0] == "LS" {
				d.list(command[1])
			} else {
				log.Println("Please enter valid command!")
				continue
			}
		}
	}
}

// UDPListener ...
// Thread to listen to UDP port
// UDPListener is listening on port 4000
func (d *Daemon) UDPListener() {

	// update []membershiplist
	p := make([]byte, 4096)
	for {
		n, remoteAddr, _ := d.Connection.ReadFromUDP(p)

		if n == 0 {
			continue
		} else {
			var ret util.IncomingMessage
			err := json.Unmarshal(p[0:n], &ret)
			if err != nil {
				log.Println(err)
			}

			if ret.Cmd != "" {
				// two conditions:
				// 1. some node leaves
				// 2. contact machine joins
				d.processCmd(remoteAddr, ret.Cmd)
			} else if len(ret.Membershiplist) != 0 {
				// some nodes or contact machine send memberlist to him
				retMemlist := make([]member.Node, 10)
				for i := range ret.Membershiplist {
					retMemlist[i] = ret.Membershiplist[i]
				}
				d.updateMemberShip(remoteAddr, retMemlist)
			} else {
				if !d.Active {
					continue
				}
				// get msg from contact machine that some node joins
				id := calculateID(ret.Join.FromAddr.IP.String())
				log.Println("Recieve from introducer that node " + strconv.Itoa(id) + " joins")
				util.WriteLog(d.ID, "Receive from introducer that node "+strconv.Itoa(id)+" joins")
				d.MembershipList[id-1].Active = true
				d.MembershipList[id-1].SetHeartBeat(0)
				d.MembershipList[id-1].Fail = false
				d.updateMonitorList()
				d.updateSendList()

			}

		}

	}
}

// update membershiplist when new membershiplist from other node comes in
func (d *Daemon) updateMemberShip(soureAddr *net.UDPAddr, mlist []member.Node) {

	id := calculateID(soureAddr.IP.String())
	//log.Println("Node: Getting msg from " + strconv.Itoa(id))
	// accept membershiplist from contact machine and find its sender
	if id == contactID && soureAddr.Port == contactSender {
		log.Println("node " + strconv.Itoa(d.ID) + " joins successfully")
		util.WriteLog(d.ID, "node "+strconv.Itoa(d.ID)+" joins successfully")

		d.Active = true
		for i := range mlist {
			if mlist[i].Active {
				d.MembershipList[i].Active = true
				d.MembershipList[i].SetHeartBeat(0)
			}
		}
		d.updateSendList()
		d.updateMonitorList()
		return
	}

	if !d.Active {
		return
	}

	// normal case when a membershiplist comes in
	for i := range mlist {
		if id == mlist[i].ID {
			//increment sender's heartbeat in current node's membershiplist
			d.MembershipList[i].SetHeartBeat(0)
			d.MembershipList[i].Active = true
			d.MembershipList[i].Fail = false
		} else {
			if !d.MembershipList[i].Fail {
				// believe in others
				if d.MembershipList[i].Active {
					if mlist[i].Fail && mlist[i].Active {
						log.Println("Receive from node " + strconv.Itoa(id) + " that node " + strconv.Itoa(i+1) + " fails")
						util.WriteLog(d.ID, "Receive from node "+strconv.Itoa(id)+" that node "+strconv.Itoa(i+1)+" fails")
					}
					if !mlist[i].Fail && !mlist[i].Active {
						log.Println("Receive from node " + strconv.Itoa(id) + " that node " + strconv.Itoa(i+1) + " leaves")
						util.WriteLog(d.ID, "Receive from node "+strconv.Itoa(id)+" that node "+strconv.Itoa(i+1)+" leaves")
					}
				} else {

					if mlist[i].Fail && mlist[i].Active {
						log.Println("Case2: Receive from node " + strconv.Itoa(id) + " that node " + strconv.Itoa(i+1) + " fails")
						util.WriteLog(d.ID, "Receive from node "+strconv.Itoa(id)+" that node "+strconv.Itoa(i+1)+" fails")
					}

					if !mlist[i].Fail && mlist[i].Active {
						log.Println("Receive from node " + strconv.Itoa(id) + " that node " + strconv.Itoa(i+1) + " joins")
						util.WriteLog(d.ID, "Receive from node "+strconv.Itoa(id)+" that node "+strconv.Itoa(i+1)+" joins")
					}
				}
				d.MembershipList[i].Active = mlist[i].Active
				d.MembershipList[i].Fail = mlist[i].Fail
			}
			// otherwise, believe in myself
		}
	}
}

// UpdateAndDisseminate ...
func (d *Daemon) UpdateAndDisseminate() {
	if !d.Active {
		return
	}
	// update status of each server in membershiplist

	for i := range d.MonitorList {
		idx := d.MonitorList[i]
		if idx != d.ID-1 && d.MembershipList[idx].Active {
			d.MembershipList[idx].UpdateHeartBeat()
			if threshold < d.MembershipList[idx].Heartbeat {
				if !d.MembershipList[idx].Fail && d.MembershipList[idx].Active {
					log.Println("Detect node " + strconv.Itoa(d.MembershipList[idx].ID) + " failure")
					d.MembershipList[idx].Fail = true
				}
			}
		}
	}

	d.updateSendList()
	d.updateMonitorList()

	b := util.FormatMemberlist(d.MembershipList)
	for i := range d.SendList {
		index := d.SendList[i]
		targetAddr := d.MembershipList[index].UDP
		util.UDPSend(&targetAddr, b)
	}
}

func (d *Daemon) joinGroup() {
	if d.Active {
		log.Println("node " + strconv.Itoa(d.ID) + " already in the group")
		util.WriteLog(d.ID, "node "+strconv.Itoa(d.ID)+" already in the group")
		return
	}
	log.Println("node " + strconv.Itoa(d.ID) + " attempt to join...")
	util.WriteLog(d.ID, "node "+strconv.Itoa(d.ID)+" attempt to join...")

	// send JOIN to conact machine
	cmd := util.Command("JOIN")

	d.MembershipList[d.ID-1].Active = true
	d.MembershipList[d.ID-1].Fail = false

	b, _ := json.Marshal(cmd)
	contactAddr := net.UDPAddr{
		IP:   net.ParseIP(serverBase),
		Port: contactListener,
	}
	util.UDPSend(&contactAddr, b)
}

func (d *Daemon) leaveGroup() {
	// send leave to its target
	cmd := util.Command("LEAVE")
	d.MembershipList[d.ID-1].Active = false
	d.MembershipList[d.ID-1].Fail = false
	d.Active = false
	b, _ := json.Marshal(cmd)

	// send to contact i'm leaving
	contactAddr := net.UDPAddr{
		IP:   net.ParseIP(serverBase),
		Port: contactListener,
	}
	util.UDPSend(&contactAddr, b)
	// send to target i'm leaving
	sendMsg := util.FormatString("LEAVE")

	for i := range d.SendList {
		index := d.SendList[i]
		targetAddr := d.MembershipList[index].UDP
		util.UDPSend(&targetAddr, sendMsg)
	}

	for i := 0; i < 10; i++ {
		d.MembershipList[i] = *member.NewMember(i+1, net.UDPAddr{IP: net.ParseIP(calculateIP(i + 1)), Port: udpport}, 0)
	}
	d.MonitorList = make([]int, 0)
	d.SendList = make([]int, 0)
}

func (d *Daemon) listGroup() {
	// show all active members
	log.Print("List active nodes:")
	for i := range d.MembershipList {
		if d.MembershipList[i].Active && !d.MembershipList[i].Fail {
			log.Print("N" + strconv.Itoa(d.MembershipList[i].ID))
		}
	}
}

func (d *Daemon) processCmd(addr *net.UDPAddr, cmd util.Command) {
	if cmd == util.Command("LEAVE") {

		ip := addr.IP.String()
		id := calculateID(ip)
		// update the state to be inactive
		d.MembershipList[id-1].Active = false
		d.MembershipList[id-1].Fail = false
		log.Println("node " + strconv.Itoa(id) + " leaves...")
		util.WriteLog(d.ID, "node "+strconv.Itoa(id)+" leaves...")
		d.updateMonitorList()
		//d.updateSendList()
	} else if cmd == util.Command("ALIVE") {
		// send to contact machine that i'm alive
		if d.Active {
			localCmd := util.Command("ACTIVE")
			b, _ := json.Marshal(localCmd)
			contactAddr := net.UDPAddr{
				IP:   net.ParseIP(serverBase),
				Port: contactListener,
			}
			util.UDPSend(&contactAddr, b)
		}
	}
}

func (d *Daemon) updateSendList() {
	id := d.ID - 1
	i := 0
	temp := id
	var senderList []int
	for {
		temp++
		if temp > 9 {
			temp = 0
		}
		if temp == id {
			break
		}
		if d.MembershipList[temp].Active {
			senderList = append(senderList, temp)
			i++
		}
		if i == 4 {
			break
		}
	}
	d.SendList = senderList
}

func (d *Daemon) updateMonitorList() {
	id := d.ID - 1
	i := 0
	temp := id
	var monitorList []int
	for {
		temp--
		if temp < 0 {
			temp = 9
		}
		if temp == id {
			break
		}
		if d.MembershipList[temp].Active {
			monitorList = append(monitorList, temp)
			i++
		}

		if i == 4 {
			break
		}
	}
	d.MonitorList = monitorList
}

func (in *Introducer) ContactProc() {
	log.Println("Contact: Start contact process on server 1")
	ip := whereAmI()
	contactAddr := net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: contactListener,
	}
	conn, err := net.ListenUDP("udp", &contactAddr)
	if err != nil {
		log.Println("UDP listen error")
	}
	defer conn.Close()

	// send udp message to all group memebers and tell contact machine is alive, if anyone is
	// alive, send back message to contact machine
	contactAliveCmd := util.Command("ALIVE")
	contactAliveIM := util.IncomingMessage{Cmd: contactAliveCmd}
	contactAliveByte, _ := json.Marshal(contactAliveIM)
	for i := range in.MembershipList {
		util.ContactUDPSend(&in.MembershipList[i].UDP, contactAliveByte)
	}
	// contact machine listening procedure
	p := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(p)
		//log.Println(p)
		if err != nil {
			log.Println("Contact get UDP message err!", err)
		}

		var ret util.Command
		err = json.Unmarshal(p[0:n], &ret)
		if err != nil {
			log.Println("Contact: Get some unknown UDP package")
		} else {
			if ret == util.Command("JOIN") {
				in.MembershipList[calculateID(remoteAddr.IP.String())-1].Active = true
				// send the membershiplist to remoteAddr
				sendMemList := util.FormatMemberlist(in.MembershipList)

				remoteAddr.Port = 4000
				util.ContactUDPSend(remoteAddr, sendMemList)

				// tell all the active nodes there's node to join
				cmd := util.Command("JOIN")
				//log.Println("Contact: send JOIN command to active nodes")
				joinCmd := util.JoinCommand{CmdType: cmd, FromAddr: remoteAddr}
				sendCmd := util.IncomingMessage{Join: joinCmd}
				b, _ := json.Marshal(sendCmd)
				for i := range in.MembershipList {
					if in.MembershipList[i].Active {
						util.ContactUDPSend(&in.MembershipList[i].UDP, b)
					}
				}

			} else if ret == util.Command("LEAVE") {
				in.MembershipList[calculateID(remoteAddr.IP.String())-1].Active = false
			} else if ret == util.Command("ACTIVE") {
				in.MembershipList[calculateID(remoteAddr.IP.String())-1].Active = true
			} else {
				log.Println("Contact: Get some unknown UDP package", ret)
			}
		}
	}
}

// Get self ID based on ip address
func whoAmI() int {
	ipaddr := whereAmI()
	return calculateID(ipaddr)
}

// Get current ip address of the machine
func whereAmI() string {
	addrs, _ := net.InterfaceAddrs()
	var ipaddr string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipaddr = ipnet.IP.String()
			}
		}
	}
	return ipaddr
}

//Map current ip address base off vm1 ip address
func calculateID(serverAddr string) int {
	addr, err := strconv.Atoi(serverAddr[12:14])
	if err != nil {
		log.Fatal(">Wrong ip Address")
	}
	base, _ := strconv.Atoi(serverBase[12:14])
	return addr - base + 1
}

//Map current id base off vm1 ip address
func calculateIP(id int) string {
	base, _ := strconv.Atoi(serverBase[12:14])
	return serverBase[0:12] + strconv.Itoa(base+id-1)
}

/****************Start of SDFS function****************/

//SDFSListener listens message from master
func (d *Daemon) SDFSListener() {
	p := make([]byte, 8192)

	for {
		n, remoteAddr, _ := d.SDFSConnection.ReadFromUDP(p)

		if n == 0 {
			continue
		} else {
			var ret util.IncomingMessage
			err := json.Unmarshal(p[0:n], &ret)
			if err != nil {
				log.Println(err)
			}
			if ret.Command.Cmd == "PUT" {
				d.Replica := make(chan []int)
				d.Replica <- ret.ReplicaList
			} else if ret.Command.Cmd == "GET" {
				d.Replica := make(chan []int)
				d.Replica <- ret.ReplicaList
			} else if ret.Command.Cmd == "DELETE" {
				deleteFile(ret.Command.SdfeFileName)
			} else if re.Command.Cmd == "LS"{
				d.Replica :=make(chan []int)
			}
		}
	}


}



func (d *Daemon) put(localFile string, sdfsFile string) {
	// rpc to get replica list
	/*
		b := util.FormatMemberlist(d.MembershipList)
		for i := range d.SendList {
			index := d.SendList[i]
			targetAddr := d.MembershipList[index].UDP
			util.UDPSend(&targetAddr, b)
		}*/

	replicaList := <-d.Replica
	for i := range replicaList {

	}

}

func (d *Daemon) get(sdfsFile string, localFile string) {
	//rpc
	replicaList := <-d.Replica
	if len(replicaList) == 0 {
		log.Println("File Not available")
		return
	}
	// rpc.getfile()
}

func (d *Daemon) delete(sdfsFile string) {
	// to delete

	//invoke rpc here
}

func (d *Daemon) store() {
	//read from daemon.File
	cmd := "ls " + sdfsDir
	lsCmd := exec.Command("bash", "-c", cmd)
	cmdOut, _ := lsCmd.StdoutPipe()
	lsCmd.Start()
	outputBytes, _ := ioutil.ReadAll(cmdOut)
	lsCmd.Wait()

	fmt.Print(string(outputBytes))
	return
}

func (d *Daemon) list(sdfsFile string) {
	// send packet to master and get metadata


	//
	replicaList:=<-d.Replica
	for i:= range replicaList{
		log.Println(replicaList[i])
	}
}

func (d *Daemon) clearSDFS() {
	cmd := "rm " + sdfsDir + "*"
	clearDir := exec.Command("bash", "-c", cmd)
	clearDir.Start()
	return
}

// Receive delete command from master
func (d *Daemon) deleteFile(sdfsFile string) {
	cmd := "rm " + sdfsFile
	delFile := exec.Command("bash", "-c", cmd)
	delFile.Start()
	return
}
