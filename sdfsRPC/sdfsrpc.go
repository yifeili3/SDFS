package sdfsrpc

type Message struct {
	cmd          string
	sdfsFileName string
}

type RPCMeta struct {
	MetaData map[string]MetaInfo
	Command  Message
}

type MetaInfo struct {
	Filename    string
	ReplicaList []int
	Timestamp   int
	FileSize    int
}

func (r *RPCMeta) getMessage(msg *Message, reply *MetaInfo) error {
	if msg.cmd == "PUT" {
		*reply = r.MetaData[msg.sdfsFileName]
	} else if msg.cmd == "GET" {
		*reply = getAvailableNode()
	} else if msg.cmd == "DELETE" {

		return
	} else if msg.cmd == "LIST" {

	}

	return nil
}

func (r *RPCMaster) getFile() error {

}

func getAvailableNode() {

}
