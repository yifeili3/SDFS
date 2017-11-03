package util

import (
	"Membership/member"
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
	fp = 0
)

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
