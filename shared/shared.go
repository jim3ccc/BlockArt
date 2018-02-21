package shared

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"log"
	"math/big"
	"net"
	"sync"
	"net/rpc"
)

type Neighbours struct {
	Mux *sync.RWMutex
	Map map[string]*rpc.Client
}

type ShapeType int

const (
	// Path shape.
	PATH ShapeType = iota

	// Circle shape (extra credit).
	// CIRCLE
)

type DeleteShapeArgs struct {
	ShapeHash   string
	ValidateNum uint8
}

type OpenCanvasArgs struct {
	Str  []byte
	R, S *big.Int
}

type AddShapeArgs struct {
	ValidateNum    uint8
	ShapeType      ShapeType
	ShapeSvgString string
	Fill           string
	Stroke         string
}

type AddShapeReply struct {
	ShapeHash    string
	BlockHash    string
	InkRemaining uint32
}

// BlockTrees and AccountTable
type TreesAndAccounts struct {
	Tree BlockTree
	Acc  AccountsTable
}

// info necessary to regester the miner
type MinerInfo struct {
	Address net.Addr
	Key     ecdsa.PublicKey
}

// Settings for a canvas in BlockArt.
type CanvasSettings struct {
	// Canvas dimensions
	CanvasXMax uint32
	CanvasYMax uint32
}

// Settings for an instance of the BlockArt project/network.
type MinerNetSettings struct {
	// Hash of the very first (empty) block in the chain.
	GenesisBlockHash string

	// The minimum number of ink miners that an ink miner should be
	// connected to. If the ink miner dips below this number, then
	// they have to retrieve more nodes from the server using
	// GetNodes().
	MinNumMinerConnections uint8

	// Mining ink reward per Op and no-Op blocks (>= 1)
	InkPerOpBlock   uint32
	InkPerNoOpBlock uint32

	// Number of milliseconds between heartbeat messages to the server.
	HeartBeat uint32

	// Proof of work difficulty: number of zeroes in prefix (>=0)
	PoWDifficultyOpBlock   uint8
	PoWDifficultyNoOpBlock uint8

	// Canvas settings
	CanvasSettings CanvasSettings
}

type InkBank struct {
	Mux  sync.RWMutex
	Bank uint32
}

// A point in a shape
type Point2d struct {
	X int
	Y int
}

// An SVG Path
type Path struct {
	D          string
	Fill       string
	Stroke     string
	VertexList []Point2d
}

/*
CanvasShape struct
- Shape = All the info of the SVG Path
- OpSig = Operation's Signature
*/
type CanvasShape struct {
	Shape Path
	Op    string
	OpSig string
}

/*
Block Struct:
- parent: the hash of the previous node in the tree
- Nounce: the Nounce used to create the proof of work for this block
- Creator: the public key of the miner that generated this block
- Operation: an unordered set of Operation for the block
*/
type Block struct {
	Noop       bool
	Parent     string
	Nonce      string
	Creator    ecdsa.PublicKey
	Operations []Operation //probably a better way to implement than an array
  InvalidOps []Operation
	Depth      int64
}

/*
Block tree struct:
- Key = md5 hash of the block, hash of [previous hash, ops, Op-sigs, generator pubkey, Nounce]
- Value = the block's data that corresponds to the hash in the key
*/
type BlockTree struct {
	Mux                *sync.RWMutex
	BlockHash          map[string]Block
	LongestChainHashes []string
	Length             int64
}

type BlockTreeInfo struct {
	BlockHash          map[string]Block
	LongestChainHashes []string
	Length             int64
}

/*
Operations struct:
- OpSig = operation's signature
- Op = string of the operation
- Owner = Public key of the art node Owner of the given Op
*/
type Operation struct {
	Delete      bool
	OpSig       string
	Op          string
	Owner       ecdsa.PublicKey
	ValidateNum uint8
	InkUsed     uint32
	Shape       Path
}

/*
Operation HashMap struct:
- key = hash of the operation
- value = data for the operation that has the corresponding hash
*/
type OperationMap struct {
	Mux    *sync.RWMutex
	OpHash map[string]Operation
}

/*
Accounts Table:
- key = public key of users
- value = our notion of their ink storage
*/
type AccountsTable struct {
	Mux      *sync.RWMutex
	Accounts map[string]int
}

func ErrorFatal(msg string, e error) {
	if e != nil {
		log.Fatal("%s, err = %s\n", msg, e.Error())
	}
}

func PubKeyToString(key ecdsa.PublicKey) string {
	return string(elliptic.Marshal(key.Curve, key.X, key.Y))
}
