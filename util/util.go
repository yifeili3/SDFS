package util

import (
	"SDFS/member"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	fp             = 0
	serverBase     = "172.22.154.132"
	masterSendport = 8002
	sdfsport       = 4008
	masterRevport  = 4010
)

type RPCMeta struct {
	ReplicaList []int
	Command     Message
	Metadata    map[string]MetaInfo
	Membership  []member.Node
}

type Message struct {
	Cmd          string
	SdfsFileName string
}

type MetaInfo struct {
	Filename    string
	ReplicaList []int
	Timestamp   time.Time
	State       int
}

// struct to be used by JSON Marshal function
type Command string

type IncomingMessage struct {
	Cmd            Command
	Membershiplist MemList
	Join           JoinCommand
}

type MemList []member.Node

type JoinCommand struct {
	CmdType  Command
	FromAddr *net.UDPAddr
}

// UDPSend is a function that send the formatted information to dest address
func UDPSend(dstAddr *net.UDPAddr, info []byte) {
	srcAddr := net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 2468,
	}

	conn, err := net.DialUDP("udp", &srcAddr, dstAddr)
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	if rand.Intn(100) <= 100*fp {
		return
	}
	conn.Write(info)
}

// ContactUDPSend is a function that send formatted information from contact process....
func ContactUDPSend(dstAddr *net.UDPAddr, info []byte) {
	srcAddr := net.UDPAddr{
		IP:   net.ParseIP("172.22.154.132"),
		Port: 4002,
	}
	conn, err := net.DialUDP("udp", &srcAddr, dstAddr)
	if err != nil {
		fmt.Println(err)
	}

	defer conn.Close()
	if rand.Intn(100) <= 100*fp {
		return
	}
	conn.Write(info)
}

func MasterUDPSend(dstAddr *net.UDPAddr, info []byte) {
	srcAddr := net.UDPAddr{
		IP:   net.ParseIP(WhereAmI()),
		Port: masterSendport,
	}
	dst := &net.UDPAddr{
		Port: sdfsport,
		IP:   dstAddr.IP,
	}
	conn, err := net.DialUDP("udp", &srcAddr, dst)
	if err != nil {
		fmt.Println("Error in master sending UDP:", err)
	}
	defer conn.Close()
	conn.Write(info)
}

// MasterMetaSend is for master to send metadata to other master candidate
func MasterMetaSend(dstAddr *net.UDPAddr, info []byte) {
	fmt.Printf("Master is sending out metadata!\n")
	srcAddr := net.UDPAddr{
		IP:   net.ParseIP(WhereAmI()),
		Port: masterSendport,
	}
	dst := &net.UDPAddr{
		Port: masterRevport,
		IP:   dstAddr.IP,
	}
	conn, err := net.DialUDP("udp", &srcAddr, dst)
	if err != nil {
		fmt.Println("Error in master sending UDPMeta:", err)
	}
	defer conn.Close()
	conn.Write(info)

}

func RPCformat(rpcmeta RPCMeta) []byte {
	b, _ := json.Marshal(rpcmeta)
	return b
}

// FormatString is a function that format the input string into []byte
func FormatString(input Command) []byte {
	sendMsg := IncomingMessage{Cmd: input}
	b, _ := json.Marshal(sendMsg)
	return b
}

// FormatMemberlist is a function that format the member list into []byte
func FormatMemberlist(input []member.Node) []byte {
	var memlist MemList
	memlist = MemList(input)
	sendMsg := IncomingMessage{Membershiplist: memlist}
	b, _ := json.Marshal(sendMsg)
	return b
}

// DecodeMsg is a function that decode the message got from UDP port
// and use unmarshal to decode the input msg
// the return type is IncomingMessage
func DecodeMsg(input []byte) IncomingMessage {
	var ret IncomingMessage
	err := json.Unmarshal(input, &ret)
	if err != nil {
		log.Println("Invalid message")
	}
	return ret
}

// WriteLog is a utility function to write log to files
func WriteLog(ID int, input string) {
	filepath := "/home/yifeili3/vm" + strconv.Itoa(ID) + ".log"
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Open log file error, cannot append content to it:", filepath)
		return
	}
	defer f.Close()
	writeTime := time.Now().String()
	f.WriteString("[" + writeTime + "] " + input + "\n")
	f.Sync()
}

// Get self ID based on ip address
func WhoAmI() int {
	ipaddr := WhereAmI()
	return CalculateID(ipaddr)
}

// Get current ip address of the machine
func WhereAmI() string {
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
func CalculateID(serverAddr string) int {
	addr, err := strconv.Atoi(serverAddr[12:14])
	if err != nil {
		log.Fatal(">Wrong ip Address")
	}
	base, _ := strconv.Atoi(serverBase[12:14])
	return addr - base + 1
}

//Map current id base off vm1 ip address
func CalculateIP(id int) string {
	base, _ := strconv.Atoi(serverBase[12:14])
	return serverBase[0:12] + strconv.Itoa(base+id-1)
}
