package master

import (
	"SDFS/util"
	"encoding/json"
	"hash/fnv"
	"log"
	"net"
	"time"
)

const (
	contactListenport = 10030
	serverBase        = "172.22.154.132"
	putPending        = 1
	putDone           = 2
	deletePending     = 3
)

func (r *RPCMeta) getMessage(msg *Message, reply *MetaInfo) error {

	if msg.cmd == "PUT" {
		*reply = r.MetaData[msg.sdfsFileName]
	} else if msg.cmd == "GET" {
		*reply = getAvailableNode()
	} else if msg.cmd == "DELETE" {

	} else if msg.cmd == "LIST" {

	} else {

	}

	return nil
}

func getAvailableNode() {

}

func newMaster() (m *Master) {
	ID := util.WhoAmI()
	ipAddr := util.WhereAmI()

	addr := net.UDPAddr{
		Port: contactListenport,
		IP:   net.ParseIP(ipAddr),
	}
	m = &Master{
		Addr:            addr,
		MemberAliveList: make([]bool, 10),
		MetaData:        make(MetaMap),
	}
	for i := 0; i < 10; i++ {
		m.MemberAliveList[i] = false
	}
	m.MemberAliveList[ID-1] = true

	return m

}

func (m *Master) UDPListener() {
	// firstly, build up the UDP listen port to listen to message
	udpAddr := m.Addr
	conn, err := net.ListenUDP("udp", &udpAddr)
	if err != nil {
		log.Println("UDP listen error")
	}
	defer conn.Close()

	// do not know if need a broadcast message to tell alive.

	// a for loop to cope with all situation:
	p := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(p)
		if err != nil {
			log.Println("Contact get UDP message err!", err)
		}
		var ret RPCMeta
		err = json.Unmarshal(p[0:n], &ret)

		if err != nil {
			log.Println("Get some unknow UDP message")
		} else {
			if len(ret.Command.cmd) != 0 {
				if ret.Command.cmd == "PUT" {

				} else if ret.Command.cmd == "GET" {

				} else if ret.Command.cmd == "LS" {

				} else if ret.Command.cmd == "DELETE" {

				} else if ret.Command.cmd == "PUTCOMFIRM" {

				} else if ret.Command.cmd == "PUTACK" {

				} else if ret.Command.cmd == "GETACK" {

				} else if ret.Command.cmd == "DELETEACK" {

				}
			} else {

			}
		}

	}

}

func (m *Master) ProcessPUTReq(remoteAddr *UDPAddr, FileName string) {
	if m.isMaster == false {
		return
	}
	// check if this file already in the metadata
	if metaInfo, exist := m.MetaData[FileName]; exist {
		// it's a update operation
		// check if this operation is in 1:00 of last call
		tNow := time.Now().Unix()
		tInterval := metaInfo.Timestamp - tNow
		if tInterval > 60 {
			metaInfo.State = putPending
			metaInfo.Timestamp = tNow
			replicaList := metaInfo.ReplicaList
			// send the message back to remoteAddr about the replicalist
			reply := &RPCMeta{
				ReplicaList: replicaList,
				Command: Message{
					cmd:          "PUT",
					sdfsFileName: FileName,
				},
			}
			b, _ := json.Marshal(reply)
			util.MasterUDPSend(remoteAddr, b)
		} else {
			// within 60s, sending back that need confirm
			reply := &RPCMeta{
				Command: Message{
					cmd:          "PUTCOMFIRM",
					sdfsFileName: FileName,
				},
			}
			b, _ := json.Marshal(reply)
			util.MasterUDPSend(remoteAddr, b)
		}

	} else {
		// calculate the file position for replica and new a new metadata pair
		repList := m.FileChord(FileName)
		m.MetaData[FileName] = &MetaInfo{
			Filename:    FileName,
			ReplicaList: repList,
			Timestamp:   time.Now().Unix(),
			State:       putPending,
		}
		// return the replist
	}
}

func (m *Master) FileChord(FileName string) []int {
	h := fnv.New32a()
	h.Write([]byte(FileName))
	hashSum := h.Sum32() % 10
	count := 0
	idx := hashSum
	ret := make([]int, 3)
	for count < 3 {
		if m.MemberAliveList[idx] == true {
			ret[count] = idx
			idx += 1
			count += 1
			if idx == 10 {
				idx = 0
			}
		}
	}
	return ret

}
