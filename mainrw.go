package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"shareReadWrite/shareReadWrite"
	"strings"
)

func main() {
	fmt.Println("start")
	rpcs := rpc.NewServer()
	rpcs.Register(shareReadWrite.NewNode("localhost:9876", "localhost:10030"))
	l, err := net.Listen("unix", ":9876")
	fmt.Println("listen")
	if err != nil {
		log.Fatal("listen error:", err)

	}

	go func() {
		for {
			conn, err := l.Accept()
			if err == nil {
				fmt.Println("RPC server accept something")
				go func() {
					rpcs.ServeConn(conn)
					conn.Close()
				}()
			} else {
				fmt.Println("RPC server accept err")
				break
			}
		}
		fmt.Println("RPC server registeration end")
	}()

	fmt.Println("start go")
	// go rpc.Accept(l)
	go HandleStdIn()
	// go serveHttp(l)

	for {
		fmt.Println("block main func")
	}

}

// HandleStdIn ...
// Thread to wait for standard input
func HandleStdIn() {
	var input string
	// JOIN LEAVE LIST LISTID
	fmt.Println("handlestdin")
	for {
		fmt.Scanf("%q", &input)
		fmt.Print("intput is" + input)
		cmdLine := strings.Split(input, " ")
		cmd := cmdLine[0]
		fmt.Println(cmd)
		if cmd == "GET" {
			fmt.Print(cmdLine)
			localFilename := cmdLine[1]
			distFilename := cmdLine[2]
			rpcGetFile(localFilename, distFilename)

		} else {
			log.Println("Command not found! Please enter valid command.")
			continue
		}
	}
}

func rpcGetFile(localFilename string, distFilename string) {
	fmt.Println("rpcGetFile")
	client, err := rpc.Dial("unix", "172.22.154.132:9876")
	fmt.Println("rpcdial")
	// fmt.Println(err)
	if err != nil {
		log.Printf(">Server dialing error")
		return
	}
	// getFileContent := shareReadWrite.NewNode("localhost:9876", "localhost:10030")
	var reply string
	err = client.Call("Node.ReadLocalFile", distFilename, &reply)
	checkError(err)
	// fmt.Println("The reply is:" + reply)

	n := &shareReadWrite.Node{}
	cmd := &shareReadWrite.WriteCmd{File: localFilename, Input: reply}
	n.WriteLocalFIle(*cmd, &reply)
	fmt.Println("write file success")

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
