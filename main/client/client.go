package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testplayground/chain"
	"time"
)

func main() {
	var numclients int
	var craq bool
	var filePath string
	flag.StringVar(&filePath, "in", "", "Input the address file")
	flag.IntVar(&numclients, "nc", 1, "Number of clients to configure")
	flag.BoolVar(&craq, "craq", false, "If set to true reading from non-tail is allowed")
	flag.Parse()

	//Create array of servers from input
	var numservers int = -1
	//Servers := make([]string, 0) //all servers including client VM
	addressFile, err := os.Open(filePath)
	chain.Check(err)
	defer addressFile.Close()
	scanner := bufio.NewScanner(addressFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		chain.Addrs = append(chain.Addrs, scanner.Text())
		numservers = numservers + 1
	}

	c := chain.NewClient(numservers, 1)
	//As soon as client constructed send registration req to server to store its connection
	c.RegisterClient(numservers)

	headIndex := 0
	tailIndex := numservers - 1

	rand.Seed(time.Now().UnixNano())
	//for { //	THIS FOR LOOP IS HOW YOU RUN IT FOR 60 SECONDS.
	// Create the client request
	var msg = chain.Message{Body: c.MakeOp(chain.OpPut)}
	//clientRequest := c.MakeOp(chain.OpPut)
	fmt.Printf("Client request was: %v\n", msg)

	var resp chain.Message
	if msg.Body.(chain.Operation).OperationType == chain.OpPut {
		//fmt.Println("entered this correctly")
		//clientRequest.Value = chain.randString(1)
		c.ClientWriter[headIndex].Encode(&msg) //Encode(clientRequest) before
		serverResponse := c.ServerReader[headIndex].Decode(&resp)
		if serverResponse != nil {
			log.Fatalln(serverResponse)
		}
		fmt.Printf("server response was: %v\n", resp)
	} else {
		c.ClientWriter[tailIndex].Encode(&msg) //Encode(clientRequest) before
		serverResponse := c.ServerReader[tailIndex].Decode(&resp)
		if serverResponse != nil {
			log.Fatalln(serverResponse)
		}
		fmt.Printf("server response was: %v\n", resp)
	}
	//}
}
