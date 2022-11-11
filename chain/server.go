package chain

import (
	"encoding/gob"
	"net"
	"sync"
	//"golang.org/x/exp/slices"
)

type Server struct {
	store     map[int]string
	Buffer    map[uint64]Operation //stored by msgID?
	Addr      string               //Private address to listen on
	Id        int                  //Process ID of the server
	IsHead    bool
	IsTail    bool
	mu        sync.Mutex
	AdjWriter map[int]*gob.Encoder //Store prev/next server writer
	AdjReader map[int]*gob.Decoder //Store prev/next server reader
	clients   map[int]*gob.Encoder //store client writer for all clientIDs
}

func NewServer(address string, serverid int) *Server {
	return &Server{
		store:     make(map[int]string),
		Buffer:    make(map[uint64]Operation),
		Id:        serverid,
		Addr:      address,
		IsHead:    false,
		IsTail:    false,
		AdjWriter: make(map[int]*gob.Encoder),
		AdjReader: make(map[int]*gob.Decoder),
		clients:   make(map[int]*gob.Encoder),
	}
}

func (s *Server) HandleConn(con net.Conn) {
	reader := gob.NewDecoder(con)
	var msg = &Message{}
	for {
		reader.Decode(&msg)

		switch t := msg.Body.(type) {
		case Operation:
			{
				switch t.OperationType {
				case OpPut:
					s.write(t)
				case OpGet:
					s.read(t)
				case OpDelete:
					s.delete(t)
				}
			}
		case Ack:
			{
				//received ack - delete msg from buffer
				if t.Noti == "ack" {
					s.mu.Lock()
					delete(s.Buffer, t.Msg)
					s.mu.Unlock()
					//If node hasn't spoken to predecessor before, dial and store socket for reading/writing to/from it
					if !s.IsHead {
						if s.AdjWriter[0] == nil {
							con, err := net.Dial("tcp", Addrs[s.Id-1])
							if err != nil {
								panic(err)
							}
							s.mu.Lock()
							s.AdjWriter[0] = gob.NewEncoder(con)
							s.AdjReader[1] = gob.NewDecoder(con)
							s.mu.Unlock()
						}
						s.mu.Lock()
						s.AdjWriter[0].Encode(&msg) //Send ack to previous node
						s.mu.Unlock()
					}
				}
			}
		case Registration:
			{
				s.mu.Lock()
				s.clients[t.ClientId] = gob.NewEncoder(con)
				ack := &Message{Body: Registration{ClientId: t.ClientId}}
				s.clients[t.ClientId].Encode(&ack)
				s.mu.Unlock()
			}

		}
	} //for top level for loop - permits streams of msgs to be sent vs. just 1
}

func (s *Server) write(op Operation) error {
	// Apply payload
	s.mu.Lock()
	s.store[op.Key] = op.Value

	//Store operation in buffer in case of failure to fwd propagate
	s.Buffer[op.MsgId] = op
	s.mu.Unlock()

	//If not the tail, fwd propagate msg
	if !s.IsTail {
		s.mu.Lock()
		if s.AdjWriter[1] == nil {
			con, err := net.Dial("tcp", Addrs[s.Id+1])
			if err != nil {
				panic(err)
			}
			s.AdjWriter[1] = gob.NewEncoder(con)
			s.AdjReader[0] = gob.NewDecoder(con)
		}
		var msg = &Message{Body: op}
		s.AdjWriter[1].Encode(&msg) //Send msg to next node
		s.mu.Unlock()
		return nil
	}

	//At tail, write to client connection and start back propagating ACKs
	var clientMsg = &Message{Body: Ack{Noti: op.Value, Msg: op.MsgId}}
	s.mu.Lock()
	s.clients[op.ClientId].Encode(&clientMsg)
	s.mu.Unlock()
	//If not the head, back propagate to predecessor node
	if !s.IsHead {
		var nodeAck = &Message{Body: Ack{Noti: "ack", Msg: op.MsgId, ClientId: op.ClientId}}
		s.mu.Lock()
		if s.AdjWriter[0] == nil {
			con, err := net.Dial("tcp", Addrs[s.Id-1])
			if err != nil {
				panic(err)
			}
			s.AdjWriter[0] = gob.NewEncoder(con)
		}
		s.AdjWriter[0].Encode(&nodeAck)
		s.mu.Unlock()
	}
	return nil
}

// Talks back to client for reads
func (s *Server) read(op Operation) error {
	s.mu.Lock()
	val := s.store[op.Key]
	if val == "" {
		val = "nil"
	}

	var clientMsg = &Message{Body: Ack{Noti: val, Msg: op.MsgId}}
	s.clients[op.ClientId].Encode(&clientMsg)
	s.mu.Unlock()
	return nil
}

// Return deleted key
func (s *Server) delete(op Operation) error {
	s.mu.Lock()
	delete(s.store, op.Key)
	s.Buffer[op.MsgId] = op
	s.mu.Unlock()
	//fwd propagate
	if !s.IsTail {
		if s.AdjWriter[1] == nil {
			con, err := net.Dial("tcp", Addrs[s.Id+1])
			if err != nil {
				panic(err)
			}
			s.mu.Lock()
			s.AdjWriter[1] = gob.NewEncoder(con)
			s.AdjReader[0] = gob.NewDecoder(con)
			s.mu.Unlock()
		}
		var msg = &Message{Body: op}
		s.mu.Lock()
		s.AdjWriter[1].Encode(&msg) //Send msg to next node
		s.mu.Unlock()
		return nil
	}
	//At tail, write to client connection and start back propagating ACKs
	var clientMsg = &Message{Body: Ack{Noti: op.Value, Msg: op.MsgId}}
	s.mu.Lock()
	s.clients[op.ClientId].Encode(&clientMsg)
	s.mu.Unlock()

	//If not the head, back propagate to predecessor node
	if !s.IsHead {
		var nodeAck = &Message{Body: Ack{Noti: "ack", Msg: op.MsgId, ClientId: op.ClientId}}
		s.mu.Lock()
		if s.AdjWriter[0] == nil {
			con, err := net.Dial("tcp", Addrs[s.Id-1])
			if err != nil {
				panic(err)
			}
			s.AdjWriter[0] = gob.NewEncoder(con)
		}
		s.AdjWriter[0].Encode(&nodeAck)
		s.mu.Unlock()
	}
	return nil
}
