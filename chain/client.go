package chain

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync/atomic"
)

type Client struct {
	ClientWriter []*gob.Encoder
	ServerReader []*gob.Decoder
	Id           int
	MsgId        uint64
	//Data         []Measurement
}

func NewClient(numservers, clientid int) *Client {
	return &Client{
		ClientWriter: make([]*gob.Encoder, numservers),
		ServerReader: make([]*gob.Decoder, numservers),
		MsgId:        0,
		Id:           clientid,
		//Data:         make([]Measurement, 0),
	}
}

func (c *Client) RegisterClient(numservers int) {
	//fmt.Printf("Client hit registration?")
	for s := 1; s <= numservers; s++ {
		con, err := net.Dial("tcp", Addrs[s])
		if err != nil {
			log.Fatalln(err)
		}
		c.ClientWriter[s-1] = gob.NewEncoder(con)
		c.ServerReader[s-1] = gob.NewDecoder(con)
		msg := Message{Body: Registration{ClientId: c.Id}}
		e := c.ClientWriter[s-1].Encode(&msg)
		if err != nil {
			log.Fatalln("error registering client", e)
		}
		var resp Message
		c.ServerReader[s-1].Decode(&resp)
		fmt.Printf("server %d registered client:%d\n", s-1, resp.Body.(Registration).ClientId)
	}
}

// Generate random string of length n
func randString(n int) string {
	var charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, n)
	for i := range s {
		s[i] = charset[rand.Intn(len(charset))]
	}
	return string(s)
}

// Creates an operation of the requested type
func MakeOp(req OpType, opNum *uint64) Operation { //, key int
	// Generates random key in half-open interval (1,1000] (project spec)
	minkey := 2
	maxkey := 1000
	newOp := Operation{}
	k := rand.Intn(maxkey-minkey) + minkey
	newOp.Key = k
	newOp.MsgId = atomic.AddUint64(opNum, 1)
	newOp.OperationType = req //optype
	if req == OpPut {
		newOp.Value = randString(64)
	}
	return newOp
}
