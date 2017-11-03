package shareReadWrite

import (
	"fmt"
	"io/ioutil"
	"os"
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
	fmt.Printf("Read %s success", file)
	*ret = string(fin)
	return err
}

func (n *Node) WriteLocalFIle(cmd WriteCmd, ret *string) error {
	err := ioutil.WriteFile(cmd.File, []byte(cmd.Input), 0666)
	return err
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
