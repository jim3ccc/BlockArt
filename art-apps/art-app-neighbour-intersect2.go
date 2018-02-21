/*

Usage:
go run art-app.go
*/

package main

import(
	"../blockartlib"
 	"../shared"
 	"fmt"
	"os"
 	"encoding/hex"
 	"crypto/x509"
 	"../HTML"
)

func main() {
	minerAddr := HTML.ReadFile("../minerAddr.txt")
	// private key # 4
	privKey := HTML.ReadFile("../minerPrivateKey.txt")
	privKeyBytesRestored, _ := hex.DecodeString(privKey)
	private_key, _ := x509.ParseECPrivateKey(privKeyBytesRestored)
	// Open a canvas.
	canvas, _, err := blockartlib.OpenCanvas(minerAddr, *private_key)
	checkError("open canvas: ", err)
    validateNum := uint8(2)

	// Add a line.
	_, _, _, err = canvas.AddShape(validateNum, shared.PATH, "m 100 300 l 50 -101 l 50 101 l 50 -100 l 50 100 l -100 100 l -100 -100", "transparent", "red")
	checkError("addshape: ", err)

	inkRemaining, err := canvas.GetInk()
	fmt.Println("ink remaining: ", inkRemaining)
	checkError("ink inkRemaining: ", err)

	err = canvas.GetCanvas()
	if err != nil {
		fmt.Println("8", err)
	}
	// assert ink3 > ink2

	// Close the canvas.
	_, err = canvas.CloseCanvas()
	checkError("closeCanvas: ", err)
}

// If error is non-nil, print it out and return it.
func checkError(msg string, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s, err = %s\n", msg, err.Error())
		return err
	}
	return nil
}
