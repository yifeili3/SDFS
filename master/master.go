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

type Master struct {
	MetaData        util.MetaMap
	MemberAliveList []bool
	Addr            net.UDPAddr
	IsMaster        bool
}

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
		MetaData:        make(util.MetaMap),
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
		var ret util.RPCMeta
		err = json.Unmarshal(p[0:n], &ret)

		if err != nil {
			log.Println("Get some unknow UDP message")
		} else {
			if len(ret.Command.cmd) != 0 {
				if ret.Command.cmd == "PUT" {
					m.ProcessPUTReq(&remoteAddr, ret.Command.SdfsFileName)
				} else if ret.Command.cmd == "GET" {
					m.ProcessPUTReq(&remoteAddr, ret.Command.SdfsFileName)
				} else if ret.Command.cmd == "LS" {
					m.ProcessLSReq(&remoteAddr, ret.Command.SdfsFileName)
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
			genReplyandSend(replicaList, "PUT", FileName, remoteAddr)
		} else {
			// within 60s, sending back that need confirm
			genReplyandSend(make([]int, 0), "PUTCONFIRM", FileName, remoteAddr)
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
		genReplyandSend(repList, "PUT", FileName, remoteAddr)

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
			ret[count] = idx + 1
			idx += 1
			count += 1
			if idx == 10 {
				idx = 0
			}
		}
	}
	return ret
}

func (m *Master) ProcessGETReq(remoteAddr *UDPAddr, FileName string) {
	if m.isMaster == false {
		return
	}

	if metaInfo, exist := m.MetaData[FileName]; exist {
		// if the meata info is stored in the map
		repList := metaInfo.ReplicaList
		genReplyandSend(repList, "GET", FileName, remoteAddr)
	} else {
		// this file is not in distributed system
		genReplyandSend(make([]int, 0), "GETNULL", FileName, remoteAddr)
	}
}

func (m *Master) ProcessLSReq(remoteAddr *UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	if metaInfo, exist := m.MetaData[FileName]; exist {
		repList := metaInfo.ReplicaList
		genReplyandSend(repList, "LS", FileName, remoteAddr)
	} else {
		// this file does not exist
		genReplyandSend(make([]int, 0), "LSNULL", FileName, remoteAddr)
	}
}

func (m *Master) ProcessDeleteReq(remoteAddr *UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	if metaInfo, exist := m.MetaData[FileName]; exist {
		repList := metaInfo.ReplicaList
		for {

		}
	}

}
func genReplyandSend(repList []int, cmd string, sdfsfile string, remoteAddr *UDPAddr) {
	reply := geneReply(repList, cmd, sdfsfile)
	b := util.RPCformat(*reply)
	util.MasterUDPSend(remoteAddr, b)
}

func geneReply(repList []int, cmd string, sdfsfile string) *util.RPCMeta {
	return &util.RPCMeta{
		ReplicaList: repList,
		Command: util.Message{
			Cmd:          cmd,
			SdfsFileName: sdfsfile,
		},
	}
}
