package shareReadWrite

import (
	"fmt"
	"io/ioutil"
	"os"
)

const (
	localpath = "/home/yifieli3/SDFS_local/"
	distpath  = "/home/yifeili3/SDFS_dist/"
)

type Node struct {
	Myaddress     string
	Targetaddress string
}

type WriteCmd struct {
	File  string
	Input string
}

func NewNode(myaddr string, taraddr string) *Node {
	return &Node{
		Myaddress:     myaddr,
		Targetaddress: taraddr,
	}
}

func (n *Node) ReadLocalFile(file string, ret *string) error {
	fmt.Println("ReadLocalFIle in:" + file)
	fin, err := ioutil.ReadFile(file)
	checkError(err)
	fmt.Printf("Read %s success\n", file)
	*ret = string(fin)
	return err
}

func (n *Node) WriteLocalFIle(cmd WriteCmd, ret *string) error {
	err := ioutil.WriteFile(cmd.File, []byte(cmd.Input), 0666)
	fmt.Printf("Wirte file %s success\n", cmd.File)
	*ret = "WRITE"
	return err
}

func (n *Node) DeleteFile(file string, ret *string) error {
	err := os.Remove(file)
	checkError(err)
	*ret = "DELETE"
	return err
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		//os.Exit(1)
	}
}
