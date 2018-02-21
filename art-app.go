/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run art-app.go
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import "./blockartlib"
import "./shared"
import "fmt"
import "os"
import "encoding/hex"
import "crypto/x509"
import "./HTML"
//import "time"
//import "crypto/ecdsa"

func main() {
	minerAddr := HTML.ReadFile("minerAddr.txt")
	privKey := HTML.ReadFile("minerPrivateKey.txt")

	privKeyBytesRestored, _ := hex.DecodeString(privKey)
	private_key, _ := x509.ParseECPrivateKey(privKeyBytesRestored)
	// Open a canvas.
	canvas, _, err := blockartlib.OpenCanvas(minerAddr, *private_key)
	if err != nil{
		fmt.Println("0", err)
		return
	}

    validateNum := uint8(2)

	// Add a line.
	shapeHash, _, _, err := canvas.AddShape(validateNum, shared.PATH, "M 400 400 h 50 l 50 50 v 50 l -50 50 h -50 l -50 -50 v -50 z", "pink", "black")
	if err != nil{
		fmt.Println("3", err)
	}

	// Add another line.
	_, _, _, err = canvas.AddShape(validateNum, shared.PATH, "M 550 400 h 50 l 50 50 v 50 l -50 50 h -50 l -50 -50 v -50 z", "pink", "black")
	if err != nil{
		fmt.Println("4", err)
	}

	// Add a line.
	_, _, _, err = canvas.AddShape(validateNum, shared.PATH, "m 450 400 v -300 l 25 -50 h 50 l 25 50 v 300 z", "pink", "black")
	if err != nil{
		fmt.Println("3", err)
	}

	// Delete the first line.
	_, err = canvas.DeleteShape(validateNum, shapeHash)
	if err != nil{
		fmt.Println("6", err)
	}

	_, err = canvas.DeleteShape(validateNum, shapeHash)
	if err != nil{
		fmt.Println("6", err)
	}

	err = canvas.GetCanvas()
	if err != nil {
		fmt.Println("8", err)
	}
	// assert ink3 > ink2

	// Close the canvas.
	_, err = canvas.CloseCanvas()
	if err != nil{
		fmt.Println("7", err)
	}
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
