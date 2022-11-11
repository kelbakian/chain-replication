package main

import (
	"bufio"
	"chain-replication-main/chain"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var numclients int
	var craq bool
	var filePath string
	var export bool
	var deterministic bool
	flag.StringVar(&filePath, "in", "", "Input the address file")
	flag.IntVar(&numclients, "nc", 1, "Number of clients to configure")
	flag.BoolVar(&craq, "craq", false, "If set to true reading from non-tail is allowed")
	flag.BoolVar(&export, "exp", false, "If true results will export to CSV (may take a while as clients >>")
	flag.BoolVar(&deterministic, "d", false, "If true, clients will read their own writes. If false, do random reads/writes")
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

	//Start up "numclient" Clients/workers
	clients := make([]*chain.Client, 0)
	for w := 1; w <= numclients; w++ {
		c := chain.NewClient(numservers, w)
		clients = append(clients, c)
		//As soon as client constructed send registration req to server to store its connection
		c.RegisterClient(numservers) //don't do this in goroutine
	}

	//Declare/Define head/tail index and measurement vars
	var (
		headIndex   int            = 0              //head node slice index
		tailIndex   int            = numservers - 1 //tail node slice index
		readIndex   int            = tailIndex
		avgLatency  float64        = -1 // latency
		numRequests uint64         = 0  // throughput
		MsgId       uint64         = 0
		tsec        uint64         = 60 //# of seconds to run stress test for
		wg          sync.WaitGroup      //wait for all workers to finish benchmark
		mu          sync.Mutex
	)

	//Keep track of measurements, use mutex to avoid data race when appending to slice
	var allData = make([]chain.Measurement, 0)
	//Set random seed for pseudorandom operation generation

	//This is the benchmark code. Each client needs to run in the SAME benchmark (per project spec, and also so that we can see if all clients are reading the same/latest version of the data). So, each operation should be performed by all of the clients
	rand.Seed(time.Now().UnixNano())
	for start := time.Now(); time.Since(start) < time.Second*time.Duration(tsec); {
		wg.Add(len(clients))
		for c := 1; c <= len(clients); c++ {
			go func(id int) {
				defer wg.Done()
				var reqType = chain.OpType(rand.Intn(2))
				// Create the client request
				newOp := chain.MakeOp(reqType, &MsgId)

				var data chain.Measurement
				var resp chain.Message
				data.Key = newOp.Key

				if newOp.OperationType == chain.OpPut {
					data.Op = "PUT"
					newOp.ClientId = id
					var msg = chain.Message{Body: newOp}
					data.Start = time.Now().Sub(start).Seconds()
					clients[id-1].ClientWriter[headIndex].Encode(&msg)
					serverResponse := clients[id-1].ServerReader[tailIndex].Decode(&resp)
					if serverResponse != nil {
						log.Fatalln(serverResponse)
					}
				} else {
					if craq == true {
						readIndex = rand.Intn(numservers)
					}
					data.Op = "GET"
					//Read from a random node if craq==true. Otw set to tail by default
					newOp.ClientId = id
					var msg = chain.Message{Body: newOp}
					data.Start = time.Now().Sub(start).Seconds()
					clients[id-1].ClientWriter[readIndex].Encode(&msg)
					serverResponse := clients[id-1].ServerReader[readIndex].Decode(&resp)
					if serverResponse != nil {
						log.Fatalln(serverResponse)
					}
				}
				data.Val = resp.Body.(chain.Ack).Noti
				data.End = time.Now().Sub(start).Seconds()
				data.Client = id
				data.MsgId = resp.Body.(chain.Ack).Msg
				atomic.AddUint64(&numRequests, 1)
				//Append result then loop back for another operation
				mu.Lock()
				allData = append(allData, data)
				mu.Unlock()
			}(c)
		}
		wg.Wait()
	}
	//wg.Wait()
	//Print benchmark results:
	avgLatency = float64(float64(60000) / float64(numRequests)) //60000 milliseconds per minute
	fmt.Println("60 second stress test results:")
	fmt.Printf("Average latency = %f milliseconds per request\n", avgLatency)
	fmt.Printf("Total throughput = %d requests per second\n", numRequests/tsec)

	//Export data to csv if you'd like
	if export == true {
		//Create csv file
		csvFile, err := os.Create("meas.csv")
		if err != nil {
			panic(err)
		}
		defer csvFile.Close()
		w := csv.NewWriter(csvFile)
		defer w.Flush()
		for _, row := range allData {
			//fmt.Printf("writing csv row\n")
			r := []string{row.Op, strconv.Itoa(row.Key), row.Val, fmt.Sprintf("%f", row.Start), fmt.Sprintf("%f", row.End), strconv.Itoa(row.Client), strconv.Itoa(int(row.MsgId))}
			err := w.Write(r)
			if err != nil {
				panic(err)
			}
		}
	}

}
