package master

import (
	"SDFS/member"
	"SDFS/util"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"net/rpc"
	"sort"
	"strconv"
	"time"
)

const (
	masterListener = 4010
	serverBase     = "172.22.154.132"
	putPending     = 1
	putDone        = 2
	deletePending  = 3
	repairPending  = 4
	rpcServerport  = 4004
	sdfsListener   = 4008
)

type Master struct {
	MetaData        MetaMap
	MemberAliveList []bool
	Addr            net.UDPAddr
	IsMaster        bool
	MyMaster        int
}

type MetaMap map[string]*util.MetaInfo

func NewMaster() (m *Master) {
	ID := util.WhoAmI()
	ipAddr := util.WhereAmI()

	addr := net.UDPAddr{
		Port: masterListener,
		IP:   net.ParseIP(ipAddr),
	}
	m = &Master{
		Addr:            addr,
		MemberAliveList: make([]bool, 10),
		MetaData:        make(MetaMap),
		IsMaster:        true,
	}
	for i := 0; i < 10; i++ {
		m.MemberAliveList[i] = false
	}
	m.MemberAliveList[ID-1] = true
	return m
}

func (m *Master) DisseminateMeta() {
	myID := util.WhoAmI()
	for {
		time.Sleep(time.Millisecond * 200)
		if m.IsMaster == false {
			continue
		} else {
			//fmt.Printf("My master is %d\n", m.MyMaster)
			// send message to other two backup masters
			for i := 0; i < 3; i++ {
				//fmt.Println("Finding a master target\n")
				if m.MemberAliveList[i] == true && i+1 != myID {
					//fmt.Printf("Sending metadata to %d\n", i+1)
					geneMeta(m.MetaData, i+1, "METADATA")
				}
			}
		}

	}

}

func (m *Master) UDPListener() {
	fmt.Println("Start Master UDP Listener")
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
		// *************************bug here, remote addr  wrong port
		n, remoteAddr, err := conn.ReadFromUDP(p)
		// fmt.Println("Master: " + remoteAddr.String())

		if err != nil {
			log.Println("Contact get UDP message err!", err)
		}
		if n == 0 {
			continue
		} else {
			var ret util.RPCMeta
			err = json.Unmarshal(p[0:n], &ret)
			if remoteAddr.Port == 8002 {
				//fmt.Println(len(ret.Metadata))
			}
			if err != nil {
				log.Println("Get some unknow UDP message")
			} else {
				if len(ret.Command.Cmd) != 0 {
					remoteAddr.Port = sdfsListener
					if ret.Command.Cmd == "PUT" {
						m.ProcessPUTReq(remoteAddr, ret.Command.SdfsFileName)
					} else if ret.Command.Cmd == "GET" {
						m.ProcessGETReq(remoteAddr, ret.Command.SdfsFileName)
					} else if ret.Command.Cmd == "LS" {
						m.ProcessLSReq(remoteAddr, ret.Command.SdfsFileName)
					} else if ret.Command.Cmd == "DELETE" {
						m.ProcessDeleteReq(remoteAddr, ret.Command.SdfsFileName)
					} else if ret.Command.Cmd == "PUTCONFIRM" {
						m.ProcessPUTConfirm(remoteAddr, ret.Command.SdfsFileName)
					} else if ret.Command.Cmd == "PUTACK" {
						m.ProcessPUTACK(remoteAddr, ret.Command.SdfsFileName)
					} else if ret.Command.Cmd == "FAILACK" {
						m.ProcFailRepair(remoteAddr, ret.Command.SdfsFileName)
					}
				} else if len(ret.Membership) != 0 {
					m.UpdateAlivelist(ret.Membership)
				} else if len(ret.Metadata) != 0 {
					// ths message is sent by
					m.UpdateMeta(ret.Metadata)
				}
			}
		}

	}

}

func (m *Master) UpdateMeta(Metadata map[string]util.MetaInfo) {
	//fmt.Println("Master: receive metadata!")
	if m.IsMaster == true {
		return
	} else {
		//fmt.Println("Master: receive metadata and update")
		for fileName, value := range Metadata {
			if metaInfo, exist := m.MetaData[fileName]; exist {
				metaInfo.Filename = value.Filename
				metaInfo.ReplicaList = value.ReplicaList
				metaInfo.State = value.State
				metaInfo.Timestamp = value.Timestamp
			} else {
				m.MetaData[fileName] = &util.MetaInfo{
					Filename:    value.Filename,
					ReplicaList: value.ReplicaList,
					Timestamp:   value.Timestamp,
					State:       value.State,
				}
			}

		}
	}
}

func (m *Master) ProcessPUTReq(remoteAddr *net.UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	fmt.Println("In master put")
	// check if this file already in the metadata
	if metaInfo, exist := m.MetaData[FileName]; exist {
		// it's a update operation
		// check if this operation is in 1:00 of last call
		tNow := time.Now()
		tInterval := tNow.Sub(metaInfo.Timestamp)
		log.Printf("meta's timestamp is %d, tNow is %d", metaInfo.Timestamp, tNow)
		if tInterval > time.Duration(60)*time.Second {
			metaInfo.State = putPending
			metaInfo.Timestamp = tNow
			replicaList := metaInfo.ReplicaList
			// send the message back to remoteAddr about the replicalist
			genReplyandSend(replicaList, "PUT", FileName, remoteAddr)
		} else {
			metaInfo.Timestamp = tNow
			// within 60s, sending back that need confirm
			genReplyandSend(make([]int, 0), "PUTCONFIRM", FileName, remoteAddr)
		}
		fmt.Println("In master put 1")

	} else {
		// calculate the file position for replica and new a new metadata pair
		fmt.Println("In master put 2")
		repList := m.FileChord(FileName)
		m.MetaData[FileName] = &util.MetaInfo{
			Filename:    FileName,
			ReplicaList: repList,
			Timestamp:   time.Now(),
			State:       putPending,
		}
		// return the replist
		genReplyandSend(repList, "PUT", FileName, remoteAddr)
	}
	fmt.Println("Leave master put")
}

// FileChord is to calculate the file postion given a file name
// Using hash32%10 to get the position h
// Start from the h, h-1, h-2 is the three replica's location
func (m *Master) FileChord(FileName string) []int {
	h := fnv.New32a()
	h.Write([]byte(FileName))
	hashSum := h.Sum32() % 10
	count := 0
	idx := (int(hashSum))
	ret := make([]int, 3)
	fmt.Println("idx is " + strconv.Itoa(idx))
	i := idx
	printMemberAliveList(m.MemberAliveList)
	for count < 3 {
		if m.MemberAliveList[i] == true {
			//fmt.Println(idx)
			ret[count] = i + 1
			count++
		}
		i--
		if i == idx {
			break
		}
		if i == -1 {
			i = 9
		}
	}
	for count < 3 {
		ret[count] = -1
		count++
	}
	return ret
}

func (m *Master) ProcessGETReq(remoteAddr *net.UDPAddr, FileName string) {
	if m.IsMaster == false {
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

func (m *Master) ProcessLSReq(remoteAddr *net.UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	log.Println("Received LS from node:" + FileName)
	if metaInfo, exist := m.MetaData[FileName]; exist {
		if metaInfo.State == putDone {
			repList := metaInfo.ReplicaList
			genReplyandSend(repList, "LS", FileName, remoteAddr)
		} else {
			genReplyandSend(make([]int, 0), "LSNULL", FileName, remoteAddr)
		}

	} else {
		// this file does not exist
		genReplyandSend(make([]int, 0), "LSNULL", FileName, remoteAddr)
	}
}

func (m *Master) ProcessDeleteReq(remoteAddr *net.UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	if metaInfo, exist := m.MetaData[FileName]; exist {
		repList := metaInfo.ReplicaList
		for count := 0; count < 3; count++ {
			tcpAddr := calTCP(repList[count])
			client, err := rpc.DialHTTP("tcp", tcpAddr)
			if err != nil {
				log.Printf(">Server dialing error")
				return
			}
			// getFileContent := shareReadWrite.NewNode("localhost:9876", "localhost:10030")
			var reply string
			err = client.Call("Node.DeleteFile", FileName, &reply)
			// fmt.Println("The reply is:" + reply)

		}
		delete(m.MetaData, FileName)
	} else {
		// no such file, do nothing
	}

}

func (m *Master) ProcessPUTConfirm(remoteAddr *net.UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	if metaInfo, exist := m.MetaData[FileName]; exist {
		metaInfo.State = putPending
		metaInfo.Timestamp = time.Now()
		replicaList := metaInfo.ReplicaList
		genReplyandSend(replicaList, "PUT", FileName, remoteAddr)
	} else {
		// calculate the file position for replica and new a new metadata pair
		repList := m.FileChord(FileName)
		m.MetaData[FileName] = &util.MetaInfo{
			Filename:    FileName,
			ReplicaList: repList,
			Timestamp:   time.Now(),
			State:       putPending,
		}
		// return the replist
		genReplyandSend(repList, "PUT", FileName, remoteAddr)
	}
}

func (m *Master) ProcessPUTACK(remoteAddr *net.UDPAddr, FileName string) {
	if m.IsMaster == false {
		return
	}
	if metaInfo, exist := m.MetaData[FileName]; exist {
		metaInfo.State = putDone
	} else {
		log.Println("Error, get the ack of put but no such file")
	}
}

func (m *Master) UpdateAlivelist(membership []member.Node) {
	//fmt.Println("get from node and update membership")
	masterCount := [3]int{0, 0, 0}
	var needrepair []int
	for i := range membership {
		stateBefore := m.MemberAliveList[i]
		m.MemberAliveList[i] = membership[i].Active && !membership[i].Fail
		if m.MemberAliveList[i] {
			masterCount[membership[i].CurrentMasterID-1]++
		}

		stateAfter := m.MemberAliveList[i]

		if stateBefore == true && stateAfter == false {
			// need up date file to other nodes
			needrepair = append(needrepair, i)

		}

	}
	fmt.Println(masterCount)
	if masterCount[0] >= masterCount[1] {
		if masterCount[0] >= masterCount[2] {
			m.MyMaster = 1
		} else {
			m.MyMaster = 3
		}
	} else { // 1>0
		if masterCount[1] >= masterCount[2] {
			// 1>2 , it's 1
			m.MyMaster = 2
		} else {
			// 2>1
			m.MyMaster = 3
		}
	}
	if m.MyMaster == util.WhoAmI() {
		m.IsMaster = true
	} else {
		m.IsMaster = false
	}
	fmt.Printf("Master: my master is %d, I'm %d master", m.MyMaster, m.IsMaster)
	fmt.Println(needrepair)
	if m.IsMaster == true || m.MemberAliveList[m.MyMaster-1] == false {
		for i := range needrepair {
			m.FailTransferRep(needrepair[i])
		}
	}
	// printMemberAliveList(m.MemberAliveList)
}

func (m *Master) UpdateNewMaster() {
	myID := util.WhoAmI()
	for i := 0; i < 3; i++ {
		if m.MemberAliveList[i] == true {
			m.MyMaster = i + 1
			break
		}
	}
	if m.MyMaster == myID {
		m.IsMaster = true
	}
}

func (m *Master) FailTransferRep(failIndex int) {
	fmt.Printf("Node %d's data needs to be repaired\n", failIndex+1)
	fileNames := []string{}
	for fileName, metaInfo := range m.MetaData {
		fmt.Println(fileName)
		for idx := range metaInfo.ReplicaList {
			if metaInfo.ReplicaList[idx] == failIndex+1 {
				fileNames = append(fileNames, fileName)
				metaInfo.ReplicaList[idx] = -1
				metaInfo.State = repairPending
				metaInfo.Timestamp = time.Now()

			}
		}
	}
	// fileNames is all the file name in the node, then find an new available node for this replica
	for fileindex := range fileNames {
		ID := m.FindAvailNode(m.MetaData[fileNames[fileindex]].ReplicaList)
		fmt.Printf("File %s needs to be repaired in node %d\n", fileNames[fileindex], ID)
		ip := util.CalculateIP(ID)
		remoteAddr := &net.UDPAddr{
			IP: net.ParseIP(ip),
		}
		genReplyandSend(m.MetaData[fileNames[fileindex]].ReplicaList, "FAILREP", fileNames[fileindex], remoteAddr)
	}
}

// find a new available node for this file, {ID0, ID1, ID2}, the alive anti-clock node
func (m *Master) FindAvailNode(input []int) int {
	sort.Ints(input)
	var replicaNode []int
	for idx := range input {
		if input[idx] != -1 {
			replicaNode = append(replicaNode, input[idx])
		}
	}

	ID := replicaNode[0]
	var start int
	if ID-1 > 0 {
		start = ID - 1
	} else {
		start = 10
	}
	for {
		if m.MemberAliveList[start-1] == true {
			break
		}
	}
	if len(replicaNode) == 1 {
		return start
	}
	// see the replicaNode if it's equal to start, then find the anticlock for this node
	if start == replicaNode[1] {
		ID = replicaNode[1]
		if ID-1 > 0 {
			start = ID - 1
		} else {
			start = 10
		}
		for {
			if m.MemberAliveList[start-1] == true {
				break
			}
		}
		return start
	} else {
		return start
	}
}

func (m *Master) ProcFailRepair(remoteAddr *net.UDPAddr, FileName string) {
	ID := util.CalculateID(remoteAddr.IP.String())
	fmt.Printf("Get Message from %d, knowing has repaired file %s\n", ID, FileName)
	for i := range m.MetaData[FileName].ReplicaList {
		if m.MetaData[FileName].ReplicaList[i] == -1 {
			m.MetaData[FileName].ReplicaList[i] = ID
			break
		}
	}
	for i := range m.MetaData[FileName].ReplicaList {
		if m.MetaData[FileName].ReplicaList[i] == -1 {
			return
		}
	}
	m.MetaData[FileName].State = putDone
}

func genReplyandSend(repList []int, cmd string, sdfsfile string, remoteAddr *net.UDPAddr) {
	reply := geneReply(repList, cmd, sdfsfile)
	b := util.RPCformat(*reply)
	fmt.Println(remoteAddr.String())
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

func geneMeta(Metadata MetaMap, srcID int, cmd string) {

	metamap := make(map[string]util.MetaInfo)
	for fileName, metaInfo := range Metadata {
		metamap[fileName] = *metaInfo
	}
	rpcmsg := &util.RPCMeta{
		Metadata: metamap,
	}
	b := util.RPCformat(*rpcmsg)
	remoteDst := &net.UDPAddr{
		IP: net.ParseIP(util.CalculateIP(srcID)),
	}
	//fmt.Println("Master sending metadata to:" + remoteDst.String())
	util.MasterMetaSend(remoteDst, b)
}

func calTCP(ID int) string {
	ip := util.CalculateIP(ID)
	return ip + ":" + strconv.Itoa(rpcServerport)
}

func printMemberAliveList(in []bool) {
	fmt.Println(in)
}
