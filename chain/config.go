package chain

import (
	"encoding/gob"
)

// Generic message struct, body will contain e.g. operation, ack, client id, etc. robust to expanding implementation
type Message struct {
	Body interface{}
	//dir  int
}

type OpType int

const (
	OpPut OpType = iota
	OpGet
	OpDelete
)

type Operation struct {
	OperationType OpType
	Key           int
	Value         string
	MsgId         uint64
	ClientId      int
}

type Ack struct {
	Noti     string //either "ack" if successful, or hypothetically get a "msg failed" type notif
	Msg      uint64 //indicates which message/operation to delete from the buffer
	ClientId int    //which client's op this was
}

type Registration struct {
	ClientId int
}

func Check(e error) error {
	if e != nil {
		panic(e)
	}
	return nil
}

// Data struct for logging stress test results
type Measurement struct {
	Op     string //int
	Key    int
	Val    string
	Start  float64
	End    float64
	Client int
	MsgId  uint64
}

var Addrs []string

func init() {
	Addrs = make([]string, 0)
	gob.Register(Operation{})
	gob.Register(Message{})
	gob.Register(Registration{})
	gob.Register(Ack{})
	gob.Register(Message{Registration{}})
	gob.Register(Message{Operation{}})
	gob.Register(Message{Ack{}})
}
