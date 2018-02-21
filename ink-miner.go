package main

import (
	"./blockartlib"
	"./miners"
	"./shape"
	"./shared"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"os/exec"
	"./HTML"
	"encoding/csv"
	"bufio"
	"io"
	"strconv"
)

var outLog *log.Logger = log.New(os.Stderr, "[miner] ", log.Lshortfile|log.LUTC|log.Lmicroseconds)

//var miner = new(Miner)

type ValidatedOp struct {
  blockHash string
  valid     bool
}

type OpInfo struct {
  ValidateNum uint8
  ch          chan ValidatedOp
}

type Miner struct {
	Settings          shared.MinerNetSettings
	BlockHashMap      shared.BlockTree
	OperationHashMap  shared.OperationMap
	Ink               shared.InkBank
	Accounts          shared.AccountsTable
	PrivateKey        *ecdsa.PrivateKey
	PublicKey         *ecdsa.PublicKey
	Stop              chan bool
	BlockCh           chan shared.Block
	OpCh              chan shared.Operation
  OpSigChMap        map[string]OpInfo
  OpSigChMapMux     *sync.RWMutex
	CurrNextHash      string
	Mining            bool
	CurrOperations    []shared.Operation
  InvalidOperations []shared.Operation
	CurrOperationsMux *sync.RWMutex
	Neighbors         *[]net.Addr
}

// CONSTANTS
const MAX_OPS_PER_BLOCK = 5

//...........CALLS SERVER.............//

//TODO: MOVE GET NODES TO HELPER LIB

func exe_cmd(cmd string)string {
	fmt.Println("command is ",cmd)
	// splitting head => g++ parts => rest of the command
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	out, err := exec.Command(head,parts...).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	fmt.Printf("%s", out)
	return string(out)
}
func main() {
	miner := new(Miner)

	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	serverIpPort := os.Args[1]
	fmt.Println("serverIpPort", serverIpPort)
	public:=os.Args[2]
	fmt.Println("public key", public)
	private :=os.Args[3]
	fmt.Println("private key", private)
	HTML.CreateFile("minerPrivateKey.txt")
	HTML.WriteFile("minerPrivateKey.txt", private)
	IP := os.Args[4]
	fmt.Println("Miner IP", IP)

	privKeyBytesRestored, _ := hex.DecodeString(private)
	private_key, _ := x509.ParseECPrivateKey(privKeyBytesRestored)

	publKeyBytesRestored, _ := hex.DecodeString(public)
	pubkey, _ := x509.ParsePKIXPublicKey(publKeyBytesRestored)
	public_key := pubkey.(*ecdsa.PublicKey)

	//then register the miner to the server, fill in MinerInfo and instantialize empty MinerNetSettings
	server, err := rpc.Dial("tcp", serverIpPort)
	exitOnError("ink-miner main", err)

	ln, err := net.Listen("tcp", IP)//todo
	addr := ln.Addr()
	exitOnError("miner listener", err)
	fmt.Println("ink-miner address: ", addr)
	//HTML.CreateFile("minerAddr.txt")
	//HTML.WriteFile("minerAddr.txt", addr.String())

	Register(server, &miner.Settings, addr, public_key)
	miner.PrivateKey = private_key
	miner.PublicKey = public_key
	fmt.Println("miner.go main() mns", miner.Settings)

	// Send heartbeat to server
	go sendHeartBeatHelper(server, miner.PrivateKey.PublicKey)

	// Get list of neighbours and start listener
	nodes := getNeighbors(server, miner.PrivateKey.PublicKey)
	// init miner
	miner.CurrOperations = make([]shared.Operation, 0)
  miner.InvalidOperations = make([]shared.Operation, 0)
	miner.CurrOperationsMux = &sync.RWMutex{}
	miner.Mining = false
	// miner.CurrNextHash = miner.Settings.GenesisBlockHash
	miner.Stop = make(chan bool, 1)
	miner.BlockCh = make(chan shared.Block)//, 1000)
	miner.OpCh = make(chan shared.Operation)//, 1000)
  miner.OpSigChMap = make(map[string]OpInfo)
  miner.OpSigChMapMux = &sync.RWMutex{}
	miner.BlockHashMap = shared.BlockTree{Mux: &sync.RWMutex{}, BlockHash: make(map[string]shared.Block), LongestChainHashes: make([]string, 1), Length: 1}
	miner.BlockHashMap.LongestChainHashes[0] = miner.Settings.GenesisBlockHash
	miner.BlockHashMap.BlockHash[miner.Settings.GenesisBlockHash] = shared.Block{Depth: 1}
	miner.OperationHashMap = shared.OperationMap{Mux: &sync.RWMutex{}, OpHash: make(map[string]shared.Operation)}

	go miner.MinerRoutine()
	// go test(miner)

	// testVerifyOperation(miner)
	fmt.Println("harro1")
	go miners.StartListener(addr, &miner.BlockHashMap, &miner.Accounts, nodes, &miner.BlockCh, &miner.OpCh, miner.Settings.MinNumMinerConnections, server, &miner.PrivateKey.PublicKey, ln)

	// Setup rpc server for blockarlib
	clientApi := rpc.NewServer()
	clientApi.Register(miner)
	clientApi.HandleHTTP("/", "/debug")
  // "127.0.0.1:8080"
	clientL, err := net.Listen("tcp", IP)//todo
  fmt.Println("rpc server address for blockartlib: ", clientL.Addr())
	HTML.CreateFile("minerAddr.txt")
	HTML.WriteFile("minerAddr.txt", clientL.Addr().String())
	exitOnError("ink-miner blockartliblistnener", err)
	http.Serve(clientL, nil)

	// set miner.HaveEnoughNeighbors = 1 if #neighbors >= min, else 0

	// send heartbeats to neighbors to see if they disconnected from server

	//work with the filled MinerNetSettings...
	for {
		time.Sleep(1000 * time.Millisecond)
		fmt.Println(alive)
	}

}

func Register(server *rpc.Client, settings *shared.MinerNetSettings, addr net.Addr, public *ecdsa.PublicKey) {
	mi := shared.MinerInfo{Address: addr, Key: *public}
	err := server.Call("RServer.Register", mi, settings)
	exitOnError("ink-miner Register", err)
}

func getNeighbors(server *rpc.Client, publicKey ecdsa.PublicKey) *[]net.Addr {
	var reply *[]net.Addr
	err := server.Call("RServer.GetNodes", publicKey, &reply)
	exitOnError("ink-miner getNeighbors", err)
	fmt.Println("miner.go joinNetwork", reply)
	return reply
}

/*func serveNeighbors(ln net.Listener, clientApi *rpc.Server) {
	for {
		fmt.Println("hello3")
		conn, _ := ln.Accept()
		clientApi.ServeConn(conn)
	}
}*/

/*func isNeighbor(n net.Addr, list *[]net.Addr) bool {
	for _, b := range *list {
		if b == n {
			return true
		}
	}
	return false
}*/
var alive = 0

func sendHeartBeatHelper(server *rpc.Client, publicKey ecdsa.PublicKey) {
	for {
		alive++
		reply := sendHeartBeat(server, publicKey)
		time.Sleep(time.Duration(1500) * time.Millisecond)
		if !reply {
			fmt.Println("fuck")
			return
		}

	}
}

func sendHeartBeat(server *rpc.Client, publicKey ecdsa.PublicKey) bool {
	fmt.Println(".")
	var reply bool
	err := server.Call("RServer.HeartBeat", publicKey, &reply)
	if err != nil {
		exitOnError("ink-miner sendHeartBeat", err)
	}
	reply = true
	//fmt.Println("miner.go sendHeartBeat", reply)
	return reply
}

//........EXPORTED CALLS to blockartlib............//
func (m *Miner) InitializeLib(encryptedtoken *shared.OpenCanvasArgs, settings *shared.CanvasSettings) (err error) {
	fmt.Println("initializelib")
	//validate the privatekey
	ok := ecdsa.Verify(&m.PrivateKey.PublicKey, encryptedtoken.Str, encryptedtoken.R, encryptedtoken.S)

	if !ok {
		return blockartlib.DisconnectedError("")
		//add stuff to the tables in the client etc....
	}
	*settings = m.Settings.CanvasSettings
	fmt.Println(settings)

	return
}

func (m *Miner) MakeSVGCanvas(blank interface{}, reply *[]map[string]string) (err error) {
  svgtrees := *reply
  for _, hash := range m.BlockHashMap.LongestChainHashes {
    shapeMap := make(map[string]string)
    blocks := m.TraceBlockToGenesis(hash)

    for _, block := range blocks {
      if block.Noop {
        continue
      }
      //<Path d="d" fill="fill" stroke="stroke"
      for _, op := range block.Operations {
        if op.Delete {
          delete(shapeMap, string(op.Shape.D+op.Shape.Fill+op.Shape.Stroke))
        } else {
          str := "<Path d =\""
          str += op.Shape.D
          str += "\" fill=\""
          str += op.Shape.Fill
          str += "\" stroke=\""
          str += op.Shape.Stroke
          str += "\"/>"
          shapeMap[string(op.Shape.D+op.Shape.Fill+op.Shape.Stroke)] = str
        }
      }
    }
    svgtrees = append(svgtrees, shapeMap)
  }
  *reply = svgtrees
  return
}

func (m *Miner) AddShape(args *shared.AddShapeArgs, reply *shared.AddShapeReply) (err error) {
	fmt.Println("addshape called")
	svgshape := args.ShapeSvgString
	fill := args.Fill
	validateNum := args.ValidateNum
	stroke := args.Stroke
	//checking the svgshapestring
	ok, err := checkSVGString(svgshape, fill, args.Stroke)
	if !ok || err != nil {
		switch err.(type) {
		case blockartlib.InvalidShapeSvgStringError:
			reply.ShapeHash="1"
			err = blockartlib.InvalidShapeSvgStringError("svgshape")
		case blockartlib.ShapeSvgStringTooLongError:
			reply.ShapeHash="2"
			err = blockartlib.ShapeSvgStringTooLongError("svgshape")
		}
		return
	}
	//then check if we have enough ink
	vertexList, cost := shape.ShapeToVertexList(svgshape)


	if fill != "transparent" {

		if !(vertexList[0].X == vertexList[len(vertexList)-1].X && vertexList[0].Y == vertexList[len(vertexList)-1].Y) {
			reply.ShapeHash=  "1"
			err = blockartlib.InvalidShapeSvgStringError("svgshape")
			return
		}
		if stroke != "transparent" {
			cost += shape.GetArea(vertexList)
		} else {
			cost = shape.GetArea(vertexList)
		}
	} else {
		if stroke == "transparent" {
			reply.ShapeHash =  "1"
			return
		}
	}

	//checking if OutOfBoundsError:
	for _, v := range vertexList {
		if uint32(v.Y) > m.Settings.CanvasSettings.CanvasYMax || uint32(v.X) > m.Settings.CanvasSettings.CanvasXMax {
			reply.ShapeHash = "4"
			err = blockartlib.OutOfBoundsError{}
			return
		}
	}

	m.Ink.Mux.RLock()
	defer m.Ink.Mux.RUnlock()
	hash := selectRandomHash(m.BlockHashMap.LongestChainHashes)
	blocks := m.TraceBlockToGenesis(hash)

	// InkMap for checking if there is sufficient ink to draw new operation
	var ink uint32 = 0

	for _, block := range blocks {
		for _, op := range block.Operations {
			if (*m.PublicKey == op.Owner) {
				if (op.Delete) {
					ink += op.InkUsed
				} else {
					ink -= op.InkUsed
				}
			}
		}

		// Reward creator for generating the blocks
		if (*m.PublicKey == block.Creator) {
			if block.Noop {
				ink += m.Settings.InkPerNoOpBlock
			} else {
				ink += m.Settings.InkPerOpBlock
			}
		}
	}
	ok = uint32(cost) > ink

	if ok {
		fmt.Println("not enough ink")
		reply.BlockHash = strconv.Itoa(int(cost))
		err = blockartlib.InsufficientInkError(uint32(cost))
		return
	}

	//sending it to the pow for creation... wrap it in a block structure first
	opStrng := "<path d=\"" + svgshape + "\" stroke=\"" + stroke + "\" fill=" + fill + "\"/>" + time.Now().String() + "false"
	r, s, err := ecdsa.Sign(rand.Reader, m.PrivateKey, []byte(opStrng))
	if err != nil {
		return err
	}
	opSig := string(r.String() + " " + s.String())
	path := shared.Path{D: svgshape, Fill: fill, Stroke: stroke, VertexList: vertexList}
	op := shared.Operation{ValidateNum: validateNum, Owner: *m.PublicKey, Op: opStrng, OpSig: opSig, Delete: false, Shape: path, InkUsed: uint32(cost)}

  m.OpSigChMapMux.Lock()
  m.OpSigChMap[opSig] = OpInfo{ValidateNum: validateNum, ch: make(chan ValidatedOp)}
  m.OpSigChMapMux.Unlock()

	m.OpCh <- op
  fmt.Println("addshape op: ", op)
  miners.PublishOperation(op, make(map[string]int))
	shapeHash := opSig

	reply.ShapeHash = shapeHash
	m.Ink.Mux.RLock()
	defer m.Ink.Mux.RUnlock()
	reply.InkRemaining = m.Ink.Bank
	//TODO: BLOCKHASH RECEIVED FROM CHANNEL AFTER OP IS VALIDATED, SHOULD HAVE THE HASH AND BLOCK
	//reply.BlockHash =
  opInfo := <- m.OpSigChMap[opSig].ch
  m.OpSigChMapMux.Lock()
  delete (m.OpSigChMap, opSig)
  m.OpSigChMapMux.Unlock()

  if opInfo.valid {
    m.OperationHashMap.Mux.Lock()
    defer m.OperationHashMap.Mux.Unlock()
    m.OperationHashMap.OpHash[shapeHash] = op
    reply.BlockHash = opInfo.blockHash
  } else {
		reply.ShapeHash = "3"
		err = blockartlib.ShapeOverlapError("ShapeOverlapError")
		return
	}
	return
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}


func checkSVGString(str, fill, stroke string) (ok bool, err error) {
	var regex *regexp.Regexp
	if fill == "transparent" {
		regex = regexp.MustCompile(`^([mM]\s(\d)+\s(\d)+)((\s[hHvV](\s-?(\d)+)?)|(\s[lL](\s-?(\d)+)(\s-?(\d)+))|(\s[mM]\s(\d)+\s(\d)+))|(\s[zZ])*(\s[zZ])?$`)
	} else {
		regex = regexp.MustCompile(`^([mM]\s(\d)+\s(\d)+)((\s[hHvV](\s-?(\d)+)?)|(\s[lL](\s-?(\d)+)(\s-?(\d)+)))*(\s[zZ])?$`)
	}
	csvFile, _ := os.Open("Colour.csv")
	reader := csv.NewReader(bufio.NewReader(csvFile))
	var colour =[]string{}
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		x:=strings.ToLower(line[1])
		i := strings.Index(x, "(")
		if i>-1{
			colour = append( colour,x[:i-1])

		}else{
			colour = append( colour,x)
		}

	}
	 if (contains(colour,fill)==false){
	 	if fill !="transparent"{
	 		//for _,X := range colour{
	 		//	fmt.Println(X)
			//}
			//fmt.Println("fill: ", fill)
	 		//os.Exit(0)
			return false, blockartlib.InvalidShapeSvgStringError("colourIS"+fill)
		}
	 }
	 if contains(colour,stroke)==false{
		 return false, blockartlib.InvalidShapeSvgStringError("colourIS"+stroke)
	 }

	//first check for valid svg shape string
	//regex = ([mM]\s(\d)+\s(\d)+)(\s[hHvVlL](\s-?(\d)+)+)+\s[zZ]
	ok = regex.MatchString(str)
	if !ok {
		return false, blockartlib.InvalidShapeSvgStringError(str)
	}
	//then check if svg string too long
	regex = regexp.MustCompile(`.{0,255}`)
	ok = regex.MatchString(str)
	if !ok {
		return false, blockartlib.ShapeSvgStringTooLongError(str)
	}
	return true, nil
}

func (m *Miner) GetInk(blank interface{}, inkRemaining *uint32) (err error) {
	hash := selectRandomHash(m.BlockHashMap.LongestChainHashes)

	blocks := m.TraceBlockToGenesis(hash)

	// InkMap for checking if there is sufficient ink to draw new operation
	var ink uint32 = 0

	for _, block := range blocks {
		for _, op := range block.Operations {
			if (*m.PublicKey == op.Owner) {
				if (op.Delete) {
					ink += op.InkUsed
				} else {
					ink -= op.InkUsed
				}
			}
		}

		// Reward creator for generating the blocks
		if (*m.PublicKey == block.Creator) {
			if block.Noop {
				ink += m.Settings.InkPerNoOpBlock
			} else {
				ink += m.Settings.InkPerOpBlock
			}
		}
	}

	*inkRemaining = ink

	return
}

func (m *Miner) GetSVGShape(shapeHash string, svgString *string) (err error) {
	m.OperationHashMap.Mux.RLock()
	defer m.OperationHashMap.Mux.RUnlock()
	operation, ok := m.OperationHashMap.OpHash[shapeHash]
	if !ok {
		*svgString = "0"
		err = blockartlib.InvalidShapeHashError(shapeHash)
		return
	}
	//if doesn't return anyhting return invalid shapehash
	*svgString = operation.Op
	return
}

func (m *Miner) GetGenesisBlock(blank interface{}, blockHash *string) (err error) {
	*blockHash = m.Settings.GenesisBlockHash
	return
}

func (m *Miner) DeleteShape(args shared.DeleteShapeArgs, inkRemaining *uint32) (err error) {
	//TODO: look in m.OperationHashMap and check for the shapehash in args, then grab it, create a new op which is delete for op
	//then do what we do in addshape to send the op and wait for the blockhash etc...
	m.OperationHashMap.Mux.RLock()
	shape, ok := m.OperationHashMap.OpHash[args.ShapeHash]
	m.OperationHashMap.Mux.RUnlock()
	if !ok {
		*inkRemaining = uint32(0)
		err = blockartlib.ShapeOwnerError("bad owner")
		return
	}
	op := shape
	opStrng := "<path d=\"" + op.Shape.D + "\" stroke=\"" + op.Shape.Stroke + "\" fill=" + op.Shape.Fill + "\"/>" + time.Now().String() + "true"
	r, s, err := ecdsa.Sign(rand.Reader, m.PrivateKey, []byte(opStrng))
	if err != nil {
		return
	}
	opSig := string(r.String() + " " + s.String())
	op.OpSig = opSig
	op.Op = opStrng
	op.Delete = true
	op.ValidateNum = args.ValidateNum
	m.OpCh <- op
	fmt.Println("deleteshape op: ", op)
  miners.PublishOperation(op, make(map[string]int))
  m.OpSigChMapMux.Lock()
  m.OpSigChMap[opSig] = OpInfo{ValidateNum: args.ValidateNum, ch: make(chan ValidatedOp)}
  m.OpSigChMapMux.Unlock()

  <- m.OpSigChMap[opSig].ch
  m.OpSigChMapMux.Lock()
	m.OperationHashMap.Mux.Lock()
	defer m.OperationHashMap.Mux.Unlock()
  defer m.OpSigChMapMux.Unlock()
  delete(m.OpSigChMap, opSig)
	delete(m.OperationHashMap.OpHash, args.ShapeHash)
	return
}

func (m *Miner) GetShapes(blockHash string, shapeHashes *[]string) (err error) {
	m.BlockHashMap.Mux.RLock()
	defer m.BlockHashMap.Mux.RUnlock()
	block, ok := m.BlockHashMap.BlockHash[blockHash]
	//if doesn't return anyhting return invalid blockhash
	if block.Noop || !ok {
		*shapeHashes = append(*shapeHashes, "0")
		err = blockartlib.InvalidBlockHashError(blockHash)
		return
	}

	operations := block.Operations
	for _, v := range operations {
		*shapeHashes = append(*shapeHashes, v.OpSig)
	}
	return

}

func (m *Miner) GetChildren(blockhash string, blockhashes *[]string) (err error) {
	m.BlockHashMap.Mux.RLock()
	defer m.BlockHashMap.Mux.RUnlock()
	_, ok := m.BlockHashMap.BlockHash[blockhash]
	if !ok {
		*blockhashes = append(*blockhashes, "0")
		err = blockartlib.InvalidBlockHashError(blockhash)
		return
	}

	for key, value := range m.BlockHashMap.BlockHash {
		if value.Parent == blockhash {
			*blockhashes = append(*blockhashes, key)
		}
	}

	return
}

func (m *Miner) CloseCanvas(blank interface{}, inkRemaining *uint32) (err error) {

	hash := selectRandomHash(m.BlockHashMap.LongestChainHashes)

	blocks := m.TraceBlockToGenesis(hash)

	// InkMap for checking if there is sufficient ink to draw new operation
	var ink uint32 = 0

	for _, block := range blocks {
		for _, op := range block.Operations {
			if (*m.PublicKey == op.Owner) {
				if (op.Delete) {
					ink += op.InkUsed
				} else {
					ink -= op.InkUsed
				}
			}
		}

		// Reward creator for generating the blocks
		if (*m.PublicKey == block.Creator) {
			if block.Noop {
				ink += m.Settings.InkPerNoOpBlock
			} else {
				ink += m.Settings.InkPerOpBlock
			}
		}
	}

	*inkRemaining = ink
	return
}

func GetPublicPrivateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	exitOnError("ink-miner GetPublicPrivateKeyPair", err)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(priv)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)

	encodedPrivateBytes := hex.EncodeToString(privateKeyBytes)
	encodedPublicBytes := hex.EncodeToString(publicKeyBytes)
	fmt.Println("Private: ", encodedPrivateBytes, "\n")
	fmt.Println("Public: ", encodedPublicBytes)

	return priv, &priv.PublicKey
}

func pubKeyToString(key ecdsa.PublicKey) string {
	return string(elliptic.Marshal(key.Curve, key.X, key.Y))
}

func computeNonceSecretHash(op string) string {
	h := md5.New()
	h.Write([]byte(op))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}

func exitOnError(prefix string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s, err = %s\n", prefix, err.Error())
		os.Exit(1)
	}
}

/*
  We're always mining in each loop, regardless of no-op or op. If there are no operations
  the mining function will recognize that it's a no-op.
*/
func (m *Miner) MinerRoutine() {
	// if operations not empty,
	// this will be signal for noop to stop in the middle of block generation when an operation has arrived
	fmt.Println("Routine starting")
	fmt.Println("Starting off a no-op block")
	m.CurrNextHash = selectRandomHash(m.BlockHashMap.LongestChainHashes)
	m.CurrOperationsMux.Lock()
	go m.Mine()

	for {
		m.CurrOperationsMux.RLock()
		if len(m.CurrOperations) < MAX_OPS_PER_BLOCK {
			// fmt.Printf("curr number of operations: %v\n", len(m.CurrOperations))
			m.CurrOperationsMux.RUnlock()
			select {
			case block := <-m.BlockCh:
        fmt.Println("Received a block")
				// new block has arrived, validate block (need function)
				if m.validateBlock(block) {
					m.addBlockToChain(block)
					m.CurrOperationsMux.Lock()
					if m.Mining && m.CurrNextHash == block.Parent {
						fmt.Println("Currently working on the same block - restarting block")
            m.Stop <- true

            fmt.Println("stopped")
						m.CurrOperations, m.InvalidOperations, m.CurrNextHash = m.filterOperations(m.CurrOperations, block.Operations)
						go m.Mine()
						continue
					} else {
						m.CurrOperationsMux.Unlock()
					}
				} else {
          m.printBlock(block, false, false)
        }

				m.CurrOperationsMux.Lock()
				if !m.Mining {
					fmt.Println("Not currently mining a block - mining a no-op")
					m.CurrNextHash = selectRandomHash(m.BlockHashMap.LongestChainHashes)
					go m.Mine()
				} else {
					m.CurrOperationsMux.Unlock()
				}

			case op := <-m.OpCh:
				fmt.Printf("Received a new operation %v\n", op.Op)

				// check that the operations are valid against the longest chain, and return the hash of the longest chain block
				// filter out operations that aren't done
				valid, prevHash := m.ValidateOperations(m.CurrOperations, op)
				m.CurrOperationsMux.Lock()
				if valid {
					m.CurrNextHash = prevHash
					// fmt.Println("mining: ", m.Mining)
					// fmt.Println("Printing Current Operations:")
					// m.printOperations(m.CurrOperations)
					if m.Mining {

						// we've yet to finish the previous block
            m.Stop <- true
					}
					m.CurrOperations = append(m.CurrOperations, op)
					go m.Mine()
				} else {
          m.InvalidOperations = append(m.InvalidOperations, op)
          // fmt.Println("Printing Invalid Operations")
          // m.printOperations(m.InvalidOperations)
					m.CurrOperationsMux.Unlock()
				}
      default:
			}
		} else {
			m.CurrOperationsMux.RUnlock()
			// we can't take any more ops in the current mining routine fuk off
			select {
			case block := <-m.BlockCh:
				fmt.Printf("New block arrived, can't take any more ops\n")
				// new block has arrived, validate block (need function)
				if m.validateBlock(block) {
					m.addBlockToChain(block)
					m.CurrOperationsMux.Lock()
					if m.Mining && m.CurrNextHash == block.Parent {
						fmt.Println("Currently working on the same block - restarting block")
            m.Stop <- true
						m.CurrOperations, _, m.CurrNextHash = m.filterOperations(m.CurrOperations, block.Operations)
						go m.Mine()
						continue
					} else {
						m.CurrOperationsMux.Unlock()
					}
				} else {
          m.printBlock(block, false, false)
        }

				m.CurrOperationsMux.Lock()
				if !m.Mining {
					fmt.Println("Not currently mining a block - mining a no-op")
					m.CurrNextHash = selectRandomHash(m.BlockHashMap.LongestChainHashes)
					go m.Mine()
				} else {
					m.CurrOperationsMux.Unlock()
				}
			}
		}
	}
}

func (m *Miner) Mine() {
	m.Mining = true
  select {
  case <-m.Stop:
    fmt.Println("Leftover stop from last round")
  default:
    fmt.Println("no leftover stops")
  }
  fmt.Println("mining")
  m.Stop <- false
	newBlock := new(shared.Block)
	newBlock.Parent = m.CurrNextHash
	newBlock.Operations = m.CurrOperations
  newBlock.InvalidOps = m.InvalidOperations
	newBlock.Noop = len(newBlock.Operations) == 0
	newBlock.Creator = *m.PublicKey
	var difficulty uint8
	if newBlock.Noop {
		difficulty = m.Settings.PoWDifficultyNoOpBlock
	} else {
		difficulty = m.Settings.PoWDifficultyOpBlock
	}
	m.CurrOperationsMux.Unlock()
	miners.BruteForceNonce(newBlock, int64(difficulty), m.Stop)
	if newBlock.Nonce == "" {
		// likely the mining routine stopped early
		// m.CurrOperationsMux.Lock()
		// defer m.CurrOperationsMux.Unlock()
		// m.Mining = false
		return
	}
	//new block is now mined, we've already validated the options so now we can just add it to our blockCh to be handled
	// call some func to propagate this block to all other nodes
	// reset all current operations

	m.CurrOperationsMux.Lock()
  fmt.Println("Stopped mining")
	m.Mining = false
	m.CurrOperations = make([]shared.Operation, 0)
  m.InvalidOperations = make([]shared.Operation, 0)
  m.CurrOperationsMux.Unlock()
  m.BlockCh <- *newBlock
  fmt.Println("Mined")
  miners.PublishBlockCh(*newBlock, make(map[string]int))
  // m.CurrOperationsMux.Unlock()
}

// validate the newOp aganist the existing operations on top of the longest chain
// Pick one of the valid random longestChain if there are multiple valid ones
// Operation satisfies the following:
// 1) Sufficient Ink
// 2) Does not intersect with shapes created by other artnodes
// 3) Identical Signature have not been previously added
// 4) Opeartion that delete shapes refer to a shape that exists
func (m *Miner) ValidateOperations(currOps []shared.Operation, newOp shared.Operation) (bool, string) {
	// if more than 1 valid longest chain hash, use this
	var validHashes []string

	// Check if each of the longest chains are valid
	for _, hash := range m.BlockHashMap.LongestChainHashes {
		blocks := m.TraceBlockToGenesis(hash)

		// Create a Canvas based on this chain
		canvas := make(map[string][]shared.CanvasShape)
		// InkMap for checking if there is sufficient ink to draw new operation
		inkMap := make(map[string]uint32)
		// opSigMap to check identical signature have not been previously added
		opSigMap := make(map[string]uint32)

		// Update canvas, inkMap and opSigMap as we traverse each block on the blockchain
		for _, block := range blocks {
			for _, op := range block.Operations {
				m.UpdateStatesForOperation(op, &canvas, &inkMap, &opSigMap)
			}
			// Reward creator for generating the blocks
			if block.Noop {
				inkMap[shared.PubKeyToString(block.Creator)] += m.Settings.InkPerNoOpBlock
			} else {
				inkMap[shared.PubKeyToString(block.Creator)] += m.Settings.InkPerOpBlock
			}
		}

		// Update canvas, inkMap and opSigMap as we traverse currOps
		for _, op := range currOps {
			m.UpdateStatesForOperation(op, &canvas, &inkMap, &opSigMap)
		}

		// if operation is valid, add it to validHashes
		if m.ValidateOperation(newOp, canvas, inkMap, opSigMap) {
			validHashes = append(validHashes, hash)
		}
	}

	// Select a validHash Randomly if there is more than one
	selector := big.NewInt(0)
	if len(validHashes) >= 1 {
		// uniformly select a number between 0 and # valid hashes
		selector, _ = rand.Int(rand.Reader, big.NewInt(int64(len(validHashes))))
		validHash := validHashes[selector.Int64()]
		fmt.Println("ValidateOperations: Operation Valid, returning a random validhash", validHash)
		return true, validHash
	} else {
		fmt.Println("ValidateOperations: Operation Invalid, returning false", newOp)
		return false, ""
	}

}

// Check if provided op is valid against the provided canvas, inkMap and opSigMap
// Operation is valid if it satisfies the following:
// 1) Sufficient Ink
// 2) Does not intersect with shapes created by other artnodes
// 3) Identical Signature have not been previously added
// 4) Operation that delete shapes refer to a shape that exists
func (m *Miner) ValidateOperation(op shared.Operation, canvas map[string][]shared.CanvasShape, inkMap map[string]uint32, opSigMap map[string]uint32) bool {
	// 1) Check for Sufficient Ink
	if !op.Delete && inkMap[shared.PubKeyToString(op.Owner)] < op.InkUsed {
    fmt.Println(shared.PubKeyToString(op.Owner))
		fmt.Println("ValidateOperation: Not Enough Ink Br0 wdym")
		return false
	}

	// This bool is to check if deleteShape operation refer to a shape that exists
	foundShape := false

	// 2) Check for Intersection with shapes created by other artnodes
	for k, v := range canvas {
		for _, s := range v {
			// If this shape is not owned by op.Owner, we check for intersect
			if k != shared.PubKeyToString(op.Owner) {
					fmt.Println("ValidateOperation: Checking Intersection")
				if shape.IsPathIntersect(op.Shape.VertexList, s.Shape.VertexList) {
					fmt.Println("ValidateOperation: Dude these two shapes intersect stop", op.Shape.VertexList, s.Shape.VertexList)
					return false
				}
			} else { // If this shape is owned by op.Owner, we check if deleteShape operation refers to this shape
				if comparePath(op.Shape, s.Shape){
					fmt.Println("ValidateOperation: Yo Boi I found yo shape", op.Op)
					foundShape = true
				}
			}
		}
	}

	// 3) Check if identical Sig have been previously added
	_, ok := opSigMap[op.OpSig]
	if ok {
		fmt.Println("ValidateOperation: STOP trying to add the same operation")
		return false
	}

	// 4) Operation that delete shapes refer to a shape that exists
	if op.Delete && !foundShape {
		fmt.Println("ValidateOperation: Yo man I cant find the shape you are trying to delete")
    return false
	}

	fmt.Println("ValidateOperation: Operation is Valid", op.Op)
	return true
}

// Update canvas, inkMap and opSigMap for provided operation
func (m *Miner) UpdateStatesForOperation(op shared.Operation, canvas *map[string][]shared.CanvasShape, inkMap *map[string]uint32, opSigMap *map[string]uint32) {
	// Keep track of each operation ink spent and
	// Add/Remove Shape on Canvas
	if op.Delete {
		// Try to find the Shape to delete
		// Assuming we will always find the shape to delete because the chain should be valid all the way
		for k, canvasShape := range (*canvas)[shared.PubKeyToString(op.Owner)] {
			if comparePath(canvasShape.Shape, op.Shape) {
				(*canvas)[shared.PubKeyToString(op.Owner)] = append((*canvas)[shared.PubKeyToString(op.Owner)][:k], (*canvas)[shared.PubKeyToString(op.Owner)][k+1:]...)
			}
		}
		(*inkMap)[shared.PubKeyToString(op.Owner)] += op.InkUsed
		fmt.Println("UpdateStatesForOperation: Adding Ink ", op.InkUsed)
	} else {
		canvasShape := shared.CanvasShape{Shape: op.Shape, Op: op.Op, OpSig: op.OpSig}
		(*canvas)[shared.PubKeyToString(op.Owner)] = append((*canvas)[shared.PubKeyToString(op.Owner)], canvasShape)

		(*inkMap)[shared.PubKeyToString(op.Owner)] -= op.InkUsed
		fmt.Println("UpdateStatesForOperation: Removing Ink ", op.InkUsed)
	}

	// Saving seen operations, Setting an arbitrary value, doesn't matter what
	(*opSigMap)[op.OpSig] = 1
}

// Traces b to GenesisBlock and return an array of the visited blocks
func (m *Miner) TraceBlockToGenesis(blockHash string) []shared.Block {
	var blocks []shared.Block
	block, ok := m.BlockHashMap.BlockHash[blockHash]
	if ok {
		for block.Parent != "" {
			blocks = append(blocks, block)
			block = m.BlockHashMap.BlockHash[block.Parent]
		}
	}

	// We want to reverse blocks so that genesisblock is at index 0
	blocks = ReverseBlockList(blocks)

	return blocks
}

func ReverseBlockList(blocks []shared.Block) []shared.Block {
	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}
	return blocks
}

// Comapre two Paths and see if they are equal
func comparePath(p1 shared.Path, p2 shared.Path) bool {
	if p1.D != p2.D {
		return false
	}
	if p1.Fill != p2.Fill {
		return false
	}
	if p1.Stroke != p2.Stroke {
		return false
	}
	if !comparePoint2DArrays(p1.VertexList, p2.VertexList) {
		return false
	}
	return true
}

// Compare two point2dArray
func comparePoint2DArrays(a, b []shared.Point2d) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *Miner) validateBlock(block shared.Block) bool {
	// check that hashing everything + nonce returns a hash with right # of zeros
	// check that each opsig is valid, with op public key + op
	// check that prev block hash points to some good shit (existing block)
	hash := miners.ComputeHash(block)
	var difficulty uint8
	if block.Noop {
		difficulty = m.Settings.PoWDifficultyNoOpBlock
	} else {
		difficulty = m.Settings.PoWDifficultyOpBlock
	}
	if miners.CheckLastNOfHash(int(difficulty), hash) {
		for _, op := range block.Operations {
			if !verifyOperation(op) {
        fmt.Printf("Failed to verify operation: ")
        m.printOperation(op, false)
        fmt.Printf("\n")
				return false
			}
			fmt.Println("Operation is verified: ", op.Op)
		}

		_, ok := m.BlockHashMap.BlockHash[block.Parent]
		if !ok {
      fmt.Println("Cant find parent", block.Parent)
			return false
		}

		return true
	}

  fmt.Println("Block has wrong number of zeros", hash)
	return false
}

func (m *Miner) addBlockToChain(block shared.Block) {
	m.BlockHashMap.Mux.Lock()
	defer m.BlockHashMap.Mux.Unlock()
	// parent's depth in the chain
	block.Depth = m.BlockHashMap.BlockHash[block.Parent].Depth + 1


	blockHash := miners.ComputeHash(block)
	if block.Depth == m.BlockHashMap.Length {
		// it's another longest chain
		m.BlockHashMap.LongestChainHashes = append(m.BlockHashMap.LongestChainHashes, blockHash)
    m.BlockHashMap.BlockHash[blockHash] = block
    m.printBlock(block, true, true)
	} else if block.Depth > m.BlockHashMap.Length {
		// it's the new longest chain, so erase previous longest chain hashes
		m.BlockHashMap.LongestChainHashes = make([]string, 1)
		m.BlockHashMap.LongestChainHashes[0] = blockHash
		m.BlockHashMap.Length += 1
    m.BlockHashMap.BlockHash[blockHash] = block
    m.checkOpValidNum()
    m.printBlock(block, true, false)
	}

}

func (m *Miner) checkOpValidNum() {
  m.OpSigChMapMux.Lock()
  defer m.OpSigChMapMux.Unlock()
  // fmt.Printf("Seeing if there are stuff to verify\n")
  if len(m.OpSigChMap) > 0 {
    // fmt.Printf("Verifying %v operations\n", len(m.OpSigChMap))
    for opSig, _ := range m.OpSigChMap {
      // check down the blockchain ->
      validateNum := m.OpSigChMap[opSig].ValidateNum
      hash := m.BlockHashMap.LongestChainHashes[0]
      block := m.BlockHashMap.BlockHash[hash]
      for i := int64(0); i < m.BlockHashMap.Length; i+= 1 {
        // fmt.Printf("Checking the %vth block before\n", i)
        if i >= int64(validateNum) {
          // fmt.Printf("Chain is now long enough to validate, checking the %vth block before\n", i)
          // m.printBlock(block)
          if containsOp(block.Operations, opSig) {
            fmt.Println("Found operation, operation is valid")
            m.OpSigChMap[opSig].ch <- ValidatedOp{blockHash: hash, valid: true}
            return
          } else if containsOp(block.InvalidOps, opSig) {
            fmt.Println("Found operation, operation is invalid")
            m.OpSigChMap[opSig].ch <- ValidatedOp{blockHash: hash, valid: false}
            return
          }
        }

        hash = block.Parent
        block = m.BlockHashMap.BlockHash[block.Parent]
      }
    }
  }
}

func containsOp(ops []shared.Operation, opSig string) bool {
  for _, o := range ops {
    if o.OpSig == opSig {
      return true
    }
  }
  return false
}

// Returns a list of valid operations, a list of invalid operations, and hash of the longest chain
func (m *Miner) filterOperations(currOps []shared.Operation, existingOps []shared.Operation) ([]shared.Operation, []shared.Operation, string) {
	filteredOps := make([]shared.Operation, 0)
	validOps := make([]shared.Operation, 0)
	invalidOps := make([]shared.Operation, 0)

	// Filter out the overlapping Operations
	for _, cOp := range currOps {
		exists := 0
		for _, eOp := range existingOps {
			if eOp.OpSig == cOp.OpSig {
				exists += 1
			}
		}

		if exists == 0 {
			filteredOps = append(filteredOps, cOp)
		}
	}

	hash := selectRandomHash(m.BlockHashMap.LongestChainHashes)

	blocks := m.TraceBlockToGenesis(hash)

	// Create a Canvas based on this chain
	canvas := make(map[string][]shared.CanvasShape)
	// InkMap for checking if there is sufficient ink to draw new operation
	inkMap := make(map[string]uint32)
	// opSigMap to check identical signature have not been previously added
	opSigMap := make(map[string]uint32)

	// Update canvas, inkMap and opSigMap as we traverse each block on the blockchain
	for _, block := range blocks {
		for _, op := range block.Operations {
			m.UpdateStatesForOperation(op, &canvas, &inkMap, &opSigMap)
		}

		// Reward creator for generating the blocks
		if block.Noop {
			inkMap[shared.PubKeyToString(block.Creator)] += m.Settings.InkPerNoOpBlock
		} else {
			inkMap[shared.PubKeyToString(block.Creator)] += m.Settings.InkPerOpBlock
		}
	}

	// Update canvas, inkMap and opSigMap as we traverse filteredOps
	for _, op := range filteredOps {
		// if operation is valid, add it to invalidOps
		if m.ValidateOperation(op, canvas, inkMap, opSigMap) {
			validOps = append(validOps, op)
			m.UpdateStatesForOperation(op, &canvas, &inkMap, &opSigMap)
		} else { // If operation is invalid, do not update the state, add to invalidOps
			invalidOps = append(invalidOps, op)
		}
	}

	return validOps, invalidOps, hash
}


func selectRandomHash(hashes []string) string {
	selector := big.NewInt(0)
	if len(hashes) > 1 {
		// uniformly select a number between 0 and # valid hashes
		selector, _ = rand.Int(rand.Reader, big.NewInt(int64(len(hashes))))
	}

	return hashes[selector.Int64()]
}

func verifyOperation(op shared.Operation) bool {
	opsig := strings.Split(op.OpSig, " ")
	r := big.NewInt(0)
	s := big.NewInt(0)
	r.SetString(opsig[0], 10)
	s.SetString(opsig[1], 10)

	return ecdsa.Verify(&op.Owner, []byte(op.Op), r, s)
}

func (m *Miner) printBlock(block shared.Block, valid bool, branch bool) {
  hash := miners.ComputeHash(block)
  if valid {
    if branch {
      fmt.Printf("Block:[✓][ᛘ][Parent: %v, Hash: %v, Depth: %v, ", block.Parent, hash, block.Depth)
    } else {
      fmt.Printf("Block:[✓][-][Parent: %v, Hash: %v, Depth: %v, ", block.Parent, hash, block.Depth)
    }
  } else {
    fmt.Printf("Block:[✕][Parent: %v, Hash: %v, Depth: %v, ", block.Parent, hash, block.Depth)
  }
  m.printOperations(block.Operations, block.InvalidOps)
}

func (m *Miner) printOperations(validOps []shared.Operation, invalidOps []shared.Operation) {
  fmt.Printf("Operations: [")
	for _, op := range validOps {
		m.printOperation(op, true)
	}
  for _, op := range invalidOps {
    m.printOperation(op, false)
  }
  fmt.Printf("]]\n")
}

func (m *Miner) printOperation(op shared.Operation, valid bool) {
  if valid {
	 fmt.Printf("[(✓)Op:%s, InkUsed:%v, isDelete:%v ]", op.Op, op.InkUsed, op.Delete)
   } else {
    fmt.Printf("[(✕)Op:%s, InkUsed:%v, isDelete:%v ]", op.Op, op.InkUsed, op.Delete)
   }
}

/*
TESTS - THESE PURELY TEST THE BLOCKCHAIN LOGIC, NO VALIDATION
*/
// test add 2 operations, then auto generates noop blocks
func test(m *Miner) {
	fmt.Println("Initializing test")
	time.Sleep(time.Second * 1)
	fmt.Println("Sending operation")

	opStrng := "op1" + time.Now().String() + "false"
	fmt.Println(opStrng)
	r, s, err := ecdsa.Sign(rand.Reader, m.PrivateKey, []byte(opStrng))
	if err != nil {
		fmt.Println(err)
	}
	opSig := string(r.String() + " " + s.String())
	op1 := shared.Operation{ValidateNum: 2, Owner: *m.PublicKey, Op: opStrng, OpSig: opSig, Delete: false}

	opStrng = "op2" + time.Now().String() + "false"
	fmt.Println(opStrng)
	r, s, err = ecdsa.Sign(rand.Reader, m.PrivateKey, []byte(opStrng))
	if err != nil {
		fmt.Println(err)
	}
	opSig = string(r.String() + " " + s.String())
	op2 := shared.Operation{ValidateNum: 2, Owner: *m.PublicKey, Op: opStrng, OpSig: opSig, Delete: false}

	m.OpCh <- op1
	m.OpCh <- op2

}

//test sending blocks past max blocks limit for 1 block so should create 2 op blocks
func test2(m *Miner) {
	fmt.Println("Init test 2")
	time.Sleep(time.Second * 1)
	m.OpCh <- shared.Operation{Op: "Op1", OpSig: "opsig1", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op2", OpSig: "opsig2", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op3", OpSig: "opsig3", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op4", OpSig: "opsig4", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op5", OpSig: "opsig5", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op6", OpSig: "opsig6", Owner: *m.PublicKey}
}

// testin that it handles incoming blocks when it's mining properly
// if it mines to the same parent hash, it will filter out the ops and re-mine
func test3(m *Miner) {
	fmt.Println("Init test 3")
	time.Sleep(time.Second * 1)
	operations := make([]shared.Operation, 3)
	operations[0] = shared.Operation{Op: "Op1", OpSig: "opsig1", Owner: *m.PublicKey}
	operations[1] = shared.Operation{Op: "Op2", OpSig: "opsig2", Owner: *m.PublicKey}
	operations[2] = shared.Operation{Op: "Op3", OpSig: "opsig3", Owner: *m.PublicKey}

	m.OpCh <- shared.Operation{Op: "Op1", OpSig: "opsig1", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op2", OpSig: "opsig2", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op3", OpSig: "opsig3", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op4", OpSig: "opsig4", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op5", OpSig: "opsig5", Owner: *m.PublicKey}
	m.BlockCh <- shared.Block{Noop: false, Parent: "83218ac34c1834c26781fe4bde918ee4", Creator: *m.PublicKey, Operations: operations}
	m.OpCh <- shared.Operation{Op: "Op6", OpSig: "opsig6", Owner: *m.PublicKey}
	time.Sleep(time.Second * 3)

}

//
func test4(m *Miner) {
	fmt.Println("Init test 3")
	time.Sleep(time.Second * 1)
	m.OpCh <- shared.Operation{Op: "Op1", OpSig: "opsig1", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op2", OpSig: "opsig2", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op3", OpSig: "opsig3", Owner: *m.PublicKey}
	time.Sleep(time.Second * 10)
	m.OpCh <- shared.Operation{Op: "Op4", OpSig: "opsig4", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op5", OpSig: "opsig5", Owner: *m.PublicKey}
	m.OpCh <- shared.Operation{Op: "Op6", OpSig: "opsig6", Owner: *m.PublicKey}
}

func testFilteredOp(m *Miner) {
	op1 := make([]shared.Operation, 5)
	op1[0] = shared.Operation{Op: "Op2", OpSig: "opsig2", Owner: *m.PublicKey}
	op1[1] = shared.Operation{Op: "Op3", OpSig: "opsig3", Owner: *m.PublicKey}
	op1[2] = shared.Operation{Op: "Op4", OpSig: "opsig4", Owner: *m.PublicKey}
	op1[3] = shared.Operation{Op: "Op5", OpSig: "opsig5", Owner: *m.PublicKey}
	op1[4] = shared.Operation{Op: "Op6", OpSig: "opsig6", Owner: *m.PublicKey}
	op2 := make([]shared.Operation, 4)
	op2[0] = shared.Operation{Op: "Op7", OpSig: "opsig7", Owner: *m.PublicKey}
	op2[1] = shared.Operation{Op: "Op3", OpSig: "opsig3", Owner: *m.PublicKey}
	op2[2] = shared.Operation{Op: "Op8", OpSig: "opsig8", Owner: *m.PublicKey}
	op2[3] = shared.Operation{Op: "Op5", OpSig: "opsig5", Owner: *m.PublicKey}
	op3, _, hash := m.filterOperations(op1, op2)
	fmt.Println(op3, hash)
}

func testVerifyOperation(m *Miner) {
	//sending it to the pow for creation... wrap it in a block structure first
	opStrng := "garbage" + time.Now().String() + "false"
	fmt.Println(opStrng)
	r, s, err := ecdsa.Sign(rand.Reader, m.PrivateKey, []byte(opStrng))
	if err != nil {
		fmt.Println(err)
	}
	opSig := string(r.String() + " " + s.String())
	op := shared.Operation{ValidateNum: 2, Owner: *m.PublicKey, Op: opStrng, OpSig: opSig, Delete: false}

	fmt.Println("Verified operation to be: ", verifyOperation(op))

	newPrivKey, newPubKey := GetPublicPrivateKeyPair()
	opStrng = "garbage" + time.Now().String() + "true"
	fmt.Println(opStrng)
	r, s, err = ecdsa.Sign(rand.Reader, newPrivKey, []byte(opStrng))
	if err != nil {
		fmt.Println(err)
	}
	opSig = string(r.String() + " " + s.String())
	op = shared.Operation{ValidateNum: 2, Owner: *m.PublicKey, Op: opStrng, OpSig: opSig, Delete: false}
	fmt.Println("Verified operation to be: ", verifyOperation(op))

	opStrng = "garbage" + time.Now().String() + "true"
	fmt.Println(opStrng)
	r, s, err = ecdsa.Sign(rand.Reader, newPrivKey, []byte(opStrng))
	if err != nil {
		fmt.Println(err)
	}
	opSig = string(r.String() + " " + s.String())
	op = shared.Operation{ValidateNum: 2, Owner: *newPubKey, Op: opStrng, OpSig: opSig, Delete: false}
	fmt.Println("Verified operation to be: ", verifyOperation(op))
}
