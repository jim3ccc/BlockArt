package miners

import (
	"crypto/ecdsa"
)

type Neighbour struct {
	ipPort    string
	publicKey ecdsa.PublicKey
}

//to start the connection
func startConnection(ipPort string, publicKey ecdsa.PublicKey) {

}

//used to register this thread to a connection
func register() {

}

//........Exported Methods..........///
//for incoming connections to register to this thread
func Register() {

}

//to publish both operations or block to this node and for it to spread to other networks
func Publish() {

}
