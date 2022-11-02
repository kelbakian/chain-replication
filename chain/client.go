package chain

import (
	"encoding/gob"
	"log"
	"math/rand"
	"net"
	"sync/atomic"
)

type Client struct {
	ClientWriter []*gob.Encoder
	ServerReader []*gob.Decoder
	Id           int
	MsgId        int64
}

func NewClient(numservers, clientid int) *Client {
	return &Client{
		ClientWriter: make([]*gob.Encoder, numservers),
		ServerReader: make([]*gob.Decoder, numservers),
		MsgId:        0,
		Id:           clientid,
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
		msg := Message{Body: Registration{ClientId: c.Id, Addr: Addrs[0]}}
		//var msg Message; msg.Body = Registration{client: c.Id}
		//fmt.Printf("Registration msg was: %v;%v\n", msg, msg.Body)
		e := c.ClientWriter[s-1].Encode(&msg)
		if err != nil {
			log.Fatalln("error registering client", e)
		}
		var resp Message
		c.ServerReader[s-1].Decode(&resp)
		//fmt.Printf("server responded with:%v\n", resp)
	}
}

func randString(n int) string {
	var charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, n)
	for i := range s {
		s[i] = charset[rand.Intn(len(charset))]
	}
	return string(s)
}

func (c *Client) MakeOp(req OpType) Operation {
	// Generates random key in half-open interval (1,1000] (project spec)
	//Randomly generate op
	//optype := OpType(rand.Intn(2)) //0-1
	minkey := 2
	maxkey := 1000
	k := rand.Intn(maxkey-minkey) + minkey
	newOp := Operation{}
	newOp.Key = k
	newOp.MsgId = atomic.AddInt64(&c.MsgId, 1)
	newOp.ClientId = c.Id
	newOp.OperationType = req //optype
	if req == OpPut {
		newOp.Value = randString(1)
	}
	return newOp
}
