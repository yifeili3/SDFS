package member

import (
	"net"
)

//Member struct ...
type Node struct {
	ID              int
	UDP             net.UDPAddr
	Heartbeat       int
	Active          bool
	Fail            bool
	CurrentMasterID int
}

//NewMember ...
func NewMember(id int, udp net.UDPAddr, heartbeat int) (m *Node) {
	m = new(Node)
	m.ID = id
	m.UDP = udp
	m.Heartbeat = heartbeat
	m.Active = false
	m.Fail = false
	m.CurrentMasterID = 1
	return m
}

//UpdateHeartBeat ...
func (n *Node) UpdateHeartBeat() {
	n.Heartbeat++
}

//SetHeartBeat ...
func (n *Node) SetHeartBeat(heartbeat int) {
	n.Heartbeat = heartbeat
}
