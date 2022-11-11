package main

import (
	"bufio"
	"chain-replication-main/chain"
	"flag"
	"log"
	"net"
	"os"
)

func main() {
	var id int
	var head bool
	var tail bool
	var filePath string
	flag.StringVar(&filePath, "in", "", "Input the address file")
	//var addr string
	//flag.StringVar(&addr, "a", ":5051", "Input the server address or port# if local")
	flag.IntVar(&id, "id", 1, "Process ID")
	flag.BoolVar(&head, "h", false, "Set to true if this server is the leader/head")
	flag.BoolVar(&tail, "t", false, "Set to true if this server is the tail node")
	flag.Parse()

	//Read address file
	addressFile, err := os.Open(filePath)
	chain.Check(err)
	defer addressFile.Close()
	scanner := bufio.NewScanner(addressFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		chain.Addrs = append(chain.Addrs, scanner.Text())
	}

	s := chain.NewServer(chain.Addrs[id], id)
	if head == true {
		s.IsHead = true
	}
	if tail == true {
		s.IsTail = true
	}

	listener, err := net.Listen("tcp", chain.Addrs[id])
	if err != nil {
		log.Fatalln(err)
	}

	for {
		con, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		//Handles both client and server connections
		go s.HandleConn(con)
	}
}
