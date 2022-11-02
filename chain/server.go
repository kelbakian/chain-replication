package chain

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	//"golang.org/x/exp/slices"
)

type Server struct {
	store     map[int]string
	Buffer    map[int64]Operation //stored by msgID?
	Addr      string              //Private address to listen on
	Id        int                 //Process ID of the server
	IsHead    bool
	IsTail    bool
	mu        sync.Mutex
	AdjWriter map[int]*gob.Encoder
	AdjReader map[int]*gob.Decoder
	//AdjWriter []*gob.Encoder //Store prev/next server writer
	//AdjReader []*gob.Decoder //Store prev/next server reader
	clients map[int]*gob.Encoder //store client writer
	//receiver map[int]*gob.Decoder //store client receiver
}

func NewServer(address string, serverid int) *Server {
	return &Server{
		store:     make(map[int]string),
		Buffer:    make(map[int64]Operation),
		Id:        serverid,
		Addr:      address,
		IsHead:    false,
		IsTail:    false,
		AdjWriter: make(map[int]*gob.Encoder),
		AdjReader: make(map[int]*gob.Decoder),
		//AdjWriter: make([]*gob.Encoder, 0),
		//AdjReader: make([]*gob.Decoder, 0),
		clients: make(map[int]*gob.Encoder),
		//receiver: make(map[int]*gob.Decoder),
	}
}

func (s *Server) HandleConn(con net.Conn) {
	//fmt.Printf("local addr: %v\n", con.LocalAddr().String())
	//fmt.Printf("remote addr: %v\n", con.RemoteAddr().String())
	reader := gob.NewDecoder(con)
	/*If connection came from server VMs, store there. otw know it came from a client, so store there
	Hypothetically for this project could check the singular client VM, but more robust this way in real use case*/
	//fmt.Printf("Checking Addrs[] is correct: %v\n", Addrs)
	// create blank operation object
	//op := &Operation{}
	var msg = &Message{}
	for {
		// Waitfng for the client request
		//clientReader.Decode(op)
		reader.Decode(&msg)
		fmt.Printf("decoded msg: %v\n", msg.Body)
		/*if slices.Contains(Addrs[1:], con.RemoteAddr().String()) {
			if msg.dir == 1 { //moving forward
				s.AdjWriter[0] = gob.NewEncoder(con)
				s.AdjReader[1] = reader
			} else {
				s.AdjWriter[1] = gob
			}
		}*/
		//Handle client request
		//clientResponder := gob.NewEncoder(con)

		//if reflect.TypeOf(msg.Body) == Operation {
		switch t := msg.Body.(type) {
		case Operation:
			{
				switch t.OperationType {
				case OpPut:
					s.write(t)
					//s.write(op, clientResponder)
					//clientResponder.Encode(<-s.resp)
					//clientResponder.Encode(s.write(op))
				case OpGet:
					//4
					s.read(t)
					//3
					//s.clients[t.ClientId].Encode(s.store[t.Key])
					//2
					//clientResponder := gob.NewEncoder(con)
					//s.read(t, clientResponder)
					//1
					//clientResponder.Encode(<-s.resp)
					//clientResponder.Encode(s.read(op))
				case OpDelete:
					s.delete(t)
					//clientResponder.Encode(s.delete(op))
				}
			}
		case Ack:
			{
				//received ack - delete msg from buffer
				if t.Noti == "ack" {
					s.mu.Lock()
					delete(s.Buffer, t.Msg)
					s.mu.Unlock()
					if !s.IsHead {
						if s.AdjWriter[0] == nil {
							con, err := net.Dial("tcp", Addrs[s.Id-1])
							if err != nil {
								panic(err)
							}
							s.AdjWriter[0] = gob.NewEncoder(con)
							s.AdjReader[1] = gob.NewDecoder(con)
						}
						s.AdjWriter[0].Encode(msg) //Send ack to previous node
					}
				}
			}
		case Registration:
			{
				//fmt.Printf("In registration case")
				s.clients[t.ClientId] = gob.NewEncoder(con)
				ack := Message{Body: Ack{Noti: "registered", ClientId: t.ClientId}}
				s.clients[t.ClientId].Encode(&ack)
				fmt.Printf("sent registration back\n")
			}

		}
	} //for top level for loop
}

func (s *Server) write(op Operation) error {
	//func (s *Server) write(op *Operation, clientResponder *gob.Encoder) error {
	// Apply payload
	s.mu.Lock()
	s.store[op.Key] = op.Value
	//Store operation in buffer
	s.Buffer[op.MsgId] = op
	s.mu.Unlock()

	if !s.IsTail {
		if s.AdjWriter[1] == nil {
			con, err := net.Dial("tcp", Addrs[s.Id+1])
			if err != nil {
				panic(err)
			}
			s.AdjWriter[1] = gob.NewEncoder(con)
			s.AdjReader[0] = gob.NewDecoder(con)
		}
		var msg = &Message{Body: op}
		s.AdjWriter[1].Encode(&msg)
		//s.AdjWriter[1].Encode(op) //Write to next node
		//s.fwdProp(op, Addrs[s.Id+1]) //send to next server
		//wait for resp from next node
		return nil
	}
	//At tail, write to client connection and starts back propagating "ack"
	fmt.Printf("Value reached putResp?: %v\n", op.Value)
	var clientMsg = Message{Body: Ack{Noti: op.Value, ClientId: op.ClientId, Msg: op.MsgId}}
	s.clients[op.ClientId].Encode(&clientMsg)
	//s.clients[op.ClientId].Encode(op.Value)

	//If server is both head and tail, then chain length=1
	if !s.IsHead {
		var nodeAck = Message{Body: Ack{Noti: "ack", Msg: op.MsgId, ClientId: op.ClientId}}

		if s.AdjWriter[0] == nil {
			con, err := net.Dial("tcp", Addrs[s.Id-1])
			if err != nil {
				panic(err)
			}
			s.AdjWriter[0] = gob.NewEncoder(con)
		}
		s.AdjWriter[0].Encode(&nodeAck)
	}
	//s.fwdProp(op, Addrs[0])
	//s.ackResp(op, Addrs[s.Id-1])
	return nil

}

// Talks back to client for reads
func (s *Server) read(op Operation) error {
	s.mu.Lock()
	val := s.store[op.Key]
	s.mu.Unlock()
	if val == "" {
		val = "nil"
	}

	fmt.Printf("value reached getResp?: %v\n", val)
	var clientMsg = Message{Body: Ack{Noti: op.Value, ClientId: op.ClientId, Msg: op.MsgId}}
	s.clients[op.ClientId].Encode(&clientMsg)
	//clientResponder.Encode(val)
	return nil
}

// Return deleted key
func (s *Server) delete(op Operation) error {
	s.mu.Lock()
	delete(s.store, op.Key)
	s.Buffer[op.MsgId] = op
	s.mu.Unlock()
	if !s.IsTail {
		s.AdjWriter[1].Encode(op)
		return nil
	}
	//At tail, write to client + predecessors
	//clientResponder.Encode("key deleted")
	//s.ackBack(op) //acks to predecessors
	return nil
	//return "deleted key"
}

// Message acked - remove from buffer and forward to predecessor node if exists
/*func (s *Server) ackResp(op Operation, addr string) error {
	fmt.Println("reached ackback")
	//Remove msg from buffer
	s.mu.Lock()
	delete(s.Buffer, op.MsgId)
	s.mu.Unlock()
	//s.updateVersion(op)

	//Every node but head has a predecessor
	if !s.IsHead {
		prevConn, err := net.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}
		ackSender := gob.NewEncoder(prevConn)
		var ack string = "ack"
		ackSender.Encode(ack)
		//Chain[s.Id-1].ackBack(op)
		return nil
	}
	return nil
}*/

// If addrs[0] just send the op.value
/*func (s *Server) fwdProp(op Operation, addr string) error {
	nextConn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	encoder := gob.NewEncoder(nextConn)
	if addr == Addrs[0] {
		encoder.Encode(op.Value)
	} else {
		encoder.Encode(op)
	}
	return nil
}
*/
