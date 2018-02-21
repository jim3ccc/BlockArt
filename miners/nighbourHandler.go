package miners

import (
	m "../shared"
	"crypto/ecdsa"
	"fmt"
	f "fmt"
	"net"
	"net/rpc"
	"net/http"
	"os"
	"sync"
	"time"
)

type Listner struct {
	ServerAddr           string
	BlockTree            *m.BlockTree
	AccTable             *m.AccountsTable
	HaveEnoughNeighbours bool
	Address              net.Addr
	BlockCh              *chan m.Block
	OpCh                 *chan m.Operation
	Min 				 uint8
	Neighbours           m.Neighbours
}

type FloodBlock struct{
	Block 		*m.Block
	Table 		*map[string]int
}

type FloodOp struct{
	Operation	*m.Operation
	Table 		*map[string]int
}

var listener *Listner
var clientHbeatTimestamps = make(map[string]time.Time)
var hbeatMutex = &sync.RWMutex{}
var blockTreeAddr *rpc.Client

func (L *Listner) Connect(key ecdsa.PublicKey, reply *m.TreesAndAccounts) error {
	*reply = m.TreesAndAccounts{*L.BlockTree, *L.AccTable}
	return nil
}

func ListenForMiners(listener *Listner, ipaddr net.Addr, blockTree *m.BlockTree, accountTable *m.AccountsTable, Nodes *[]net.Addr, ln net.Listener) {
	server := rpc.NewServer()
	server.RegisterName("NeighbourHandler", new(Listner))
	server.HandleHTTP("/1", "/1debug")
	// l, e := net.Listen("tcp", ipaddr.String())
	f.Printf("Server started. Receiving on %s\n", ipaddr.String())
	listener.Address = ipaddr

	go serveConnection(ln)
}

func serveConnection(l net.Listener) {
		// conn, _ := l.Accept()
		// server.ServeConn(conn)
		http.Serve(l, nil)
}

//TODO add
//		BlockCh           chan shared.Block
//		OpCh              chan shared.Operation
//As input, will need later for flooding network, Add them as pointers
func StartListener(ipaddr net.Addr, blockTree *m.BlockTree, accountTable *m.AccountsTable, Nodes *[]net.Addr, BlockCh *chan m.Block, OpCh *chan m.Operation, min uint8, server *rpc.Client, publicKey *ecdsa.PublicKey, ln net.Listener) {
	// fmt.Println("neighbourHandler startListener")
	fmt.Println("startListener nodes: ", *Nodes)

	listener = new(Listner)
	listener.HaveEnoughNeighbours = uint8(len(*Nodes)) >= min
	fmt.Println("have enough neighbors: ", listener.HaveEnoughNeighbours)

	listener.BlockCh = BlockCh
	listener.OpCh = OpCh
	listener.Neighbours = m.Neighbours{Mux: &sync.RWMutex{}, Map: make(map[string]*rpc.Client)}
	listener.AccTable = accountTable
	listener.BlockTree = blockTree
	fmt.Println("caller's unupdated blocktree: ", listener.BlockTree)
	listener.Min = min
	ListenForMiners(listener, ipaddr, blockTree, accountTable, Nodes, ln)

	// for each neighbor call setUpNeighbor(miner) to establish bi-directional communication with neighbors
	for _, neighbor := range *Nodes {
		fmt.Println("size of neighbours list: ", len(*Nodes))
		fmt.Println("connecting to ", neighbor)
		neighborServer, err := rpc.DialHTTPPath("tcp", neighbor.String(), "/1")
		blockTreeAddr = neighborServer
		fmt.Println("blockTree neighborServer: ", blockTreeAddr)
		exitOnError("ink-miner rpc.Dial neighbor", err)
		setUpNeighbor(neighborServer, ipaddr, neighbor)
		fmt.Println("hi 5")
	}
	if len(*Nodes) > 0{
		//fmt.Println("getBlockTree with address: ", blockTreeAddr)
		getBlockTree(blockTreeAddr)
	}
	fmt.Println("updated blockTree", *listener.BlockTree)
	go sendHbeatToNeighbours(ipaddr, server, publicKey)
}

func sendHbeatToNeighbours(minerAddr net.Addr, serverConn *rpc.Client, publicKey *ecdsa.PublicKey){
	for {
		listener.Neighbours.Mux.Lock()
		// fmt.Print("heyadsklfjsdklfjasdf:  ")
		// fmt.Println(listener.Neighbours.Map)
		for key, neighbor := range listener.Neighbours.Map{
			// fmt.Println("090")
			err := neighbor.Call("NeighbourHandler.Hbeat", "",  new(int))
			if err != nil {
				// fmt.Print("asdfasdfasdfasdfasdfasdfasdfa")
				fmt.Println(err)
			}
			// fmt.Println("sendHbeatToNeighbours my neighbours: ", listener.Neighbours.Map)
			if err != nil{
				time.Sleep(time.Millisecond*1500)
				for k := range listener.Neighbours.Map {
				    if k == key {
				    	fmt.Println("connection lost1: ", listener.Neighbours.Map)
				        delete(listener.Neighbours.Map, k)
				        n := uint8(len(listener.Neighbours.Map))
				        fmt.Println("listener.HaveEnoughNeighbours: ", listener.HaveEnoughNeighbours)
				        fmt.Println("n: ", n)
				        if listener.HaveEnoughNeighbours && n < listener.Min{
				        	fmt.Println("getting new neighbours")
				        	listener.HaveEnoughNeighbours = false
				        	newNodes := getNeighbors(serverConn, *publicKey)
				        	fmt.Println("new nodes: ", newNodes)

				        	for _, e := range *newNodes{
				        		fmt.Println("e in NewNodes: ", e)
				        		fmt.Println("listener.Nodes: ", listener.Neighbours.Map)
				        		if _,ok := listener.Neighbours.Map[e.String()]; !ok {
				        			fmt.Println("not already a neighbor")
				        			server, err := rpc.DialHTTP("tcp", e.String())
				        			if err != nil {
				        				continue
				        			}
				        			listener.Neighbours.Map[e.String()] = server
				        		}
				        	}
				        	fmt.Println("updated neighbours: ", listener.Neighbours.Map)
				        }
				        if n >= listener.Min{
				        	listener.HaveEnoughNeighbours = true
				        }else{
				        	listener.HaveEnoughNeighbours = false
				        }
				    }
				}
			}
		}
		listener.Neighbours.Mux.Unlock()
		time.Sleep(5000*time.Millisecond)
	}
}

func getBlockTree(server *rpc.Client){
	listener.BlockTree.Mux.Lock()
	var reply m.BlockTreeInfo
	//b := m.BlockTreeInfo{BlockHash: blocktree.BlockHash, LongestChainHashes: blocktree.LongestChainHashes, Length: blocktree.Length}

	err := server.Call("NeighbourHandler.UpdateBlockTree", "", &reply)
	exitOnError("neighbourHandler getBlockTree", err)

	//fmt.Println("BlockTree got back from neighbour ", reply)
	listener.BlockTree.BlockHash = reply.BlockHash
	listener.BlockTree.LongestChainHashes = reply.LongestChainHashes
	listener.BlockTree.Length = reply.Length
	// b := m.BlockTree{BlockHash: reply.BlockHash, LongestChainHashes: reply.LongestChainHashes, Length: reply.Length}
	listener.BlockTree.Mux.Unlock()
}

func (L *Listner) UpdateBlockTree(random *string, result *m.BlockTreeInfo) (err error) {
	listener.BlockTree.Mux.Lock()
	//fmt.Println("neighbour's BlockTree: ", listener.BlockTree)


	*result = m.BlockTreeInfo{BlockHash: listener.BlockTree.BlockHash, LongestChainHashes: listener.BlockTree.LongestChainHashes, Length: listener.BlockTree.Length}
	//fmt.Println("neighbour's result: ", result)
	listener.BlockTree.Mux.Unlock()
	return
}

func getNeighbors(server *rpc.Client, publicKey ecdsa.PublicKey) *[]net.Addr {
	var reply *[]net.Addr
	err := server.Call("RServer.GetNodes", publicKey, &reply)
	exitOnError("ink-miner getNeighbors", err)
	fmt.Println("neighbourHandler.go joinNetwork", reply)
	return reply
}

func (L *Listner) Hbeat(address string, _ *int) (err error) {
	return
}


func setUpNeighbor(server *rpc.Client, us, neighbor net.Addr) *string {
	var reply *string
	listener.Neighbours.Mux.Lock()
	listener.Neighbours.Map[neighbor.String()] = server
	listener.Neighbours.Mux.Unlock()
	err := server.Call("NeighbourHandler.SetupBiDirectional", &us, &reply)
	exitOnError("ink-miner setUpNeighbor", err)
	fmt.Println("neighbourHandler.go setUpNeighbor", reply)
	return reply
}

func (L *Listner) SetupBiDirectional(neighborAddr *net.Addr, settings *string) (err error) {
	listener.Neighbours.Mux.Lock()
	fmt.Println("asdfasdfasdfadfsfasdfasdfasdfa")
	fmt.Println("listener.Neighbours", listener.Neighbours.Map)
	server, err := rpc.DialHTTPPath("tcp", (*neighborAddr).String(), "/1")
	if err != nil {
		fmt.Println(err)
	}
	listener.Neighbours.Map[(*neighborAddr).String()] = server
	fmt.Println("added new neighbor: ", *neighborAddr)
	fmt.Println("neighborAddr: ", neighborAddr)
	fmt.Println("list of neighbors: ", listener.Neighbours.Map)
	listener.Neighbours.Mux.Unlock()

	return
}


func PublishOperation(op  m.Operation, m map[string]int) (err error) {
	listener.Neighbours.Mux.RLock()
	defer listener.Neighbours.Mux.RUnlock()
	// fmt.Println("neighbourHandler PublishOperation", listener.Neighbours.Map)
	for addr, _ := range listener.Neighbours.Map {
		m[addr] += 1
	}
	m[listener.Address.String()] += 1
	floodOp := FloodOp{Operation: &op, Table: &m}

  	reply := new(bool)
	for key, val := range m {
		if val == 1 && key != listener.Address.String() {
			// fmt.Println("sent operations*************")
			server, ok := listener.Neighbours.Map[key]
			if !ok {
				continue
			}
			// fmt.Println("Before Calling ReceiveOperation server: ", key)
			//server, err := rpc.DialHTTP("tcp", addr.String())
			// fmt.Println("After Calling ReceiveOperation server: ")
			// fmt.Println("Right before Calling ReceiveOperation")
			err = server.Call("NeighbourHandler.ReceiveOperation", floodOp, &reply)
			if err != nil {
				// fmt.Println("PublishOperation failed")
				// return err
			}
		}
		if *reply == false {
			// fmt.Println("publish failed")
			// return errors.New("Publish failed")
		}
	}

	return nil
}

func PublishBlockCh(block  m.Block,  m map[string]int) (err error) {

	// fmt.Println("trying to get send block lock")
	listener.Neighbours.Mux.RLock()
	defer listener.Neighbours.Mux.RUnlock()
	for addr, _ := range listener.Neighbours.Map{
		m[addr] += 1
	}
	m[listener.Address.String()] += 1
	floodBlock := FloodBlock{Block: &block, Table: &m}
	// fmt.Println("trying to send block")
	reply := new(bool)
	// fmt.Printf("%v\n", m)
	for key, val := range m {
		// fmt.Println("In loop")
		if val == 1 && key != listener.Address.String(){
			// fmt.Println("sent my block to", key)
			server, ok := listener.Neighbours.Map[key]
			if !ok {
				fmt.Println("neighour doesn't exist", key)
				continue
			}
			err = server.Call("NeighbourHandler.ReceiveBlock", floodBlock, &reply)
			if err != nil {
				fmt.Println(err)
				// return err
			}
		} else {
			fmt.Printf("Not sending to %s, already sent\n", key)
		}
		if *reply == false {
			fmt.Println("publish failed")
			// return errors.New("Publish failed")
		}
	}
	return err
}

//No validation lel
func (L *Listner) ReceiveBlock(blockPack FloodBlock, reply *bool) error {
	// f.Println(*listener.BlockCh)
	f.Println("received block********************")

	block := blockPack.Block
	fmt.Println("going to pass to the blockch")
	*listener.BlockCh <- *block
	f.Println("publishing to own neighbours")
	go PublishBlockCh(*blockPack.Block, *blockPack.Table)
	// f.Println(*listener.BlockCh)
	*reply = true
	return nil
}

func (L *Listner) ReceiveOperation(opPack FloodOp, reply *bool) error {
	f.Println("receiveoperations***********************")
	op := opPack.Operation
	*listener.OpCh <- *op
	go PublishOperation(*opPack.Operation, *opPack.Table)
	*reply = true
	return nil
}

func exitOnError(prefix string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s, err = %s\n", prefix, err.Error())
		os.Exit(1)
	}
}
