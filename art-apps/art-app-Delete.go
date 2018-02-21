/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run art-app.go
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import "../blockartlib"
import "../shared"
import "fmt"
import "os"
import "encoding/hex"
import "crypto/x509"
import "../HTML"
//import "crypto/ecdsa"

func main() {

	minerAddr := HTML.ReadFile("../minerAddr.txt")
	privKey := HTML.ReadFile("../minerPrivateKey.txt")
	privKeyBytesRestored, _ := hex.DecodeString(privKey)
	private_key, _ := x509.ParseECPrivateKey(privKeyBytesRestored)
	// Open a canvas.
	canvas, _, err := blockartlib.OpenCanvas(minerAddr, *private_key)
	if err != nil{
		fmt.Println("0", err)
	}

	ink, err:=canvas.GetInk()
	fmt.Println(ink)
    validateNum := uint8(2)

	shapeHash, _, _, err := canvas.AddShape(validateNum, shared.PATH, "M 53 414 L 683 153 L 33 227", "transparent", "red")
	if err != nil{
		fmt.Println("red red", err)
	}else{
		fmt.Println("red red")
	}


	err = canvas.GetCanvas()
	if err != nil {
		fmt.Println("8", err)
	}


	// Delete the first line.
	_, err = canvas.DeleteShape(validateNum, string(shapeHash))
	if err != nil{
		fmt.Println("First Delete works", err)
	}

	_, err = canvas.DeleteShape(validateNum, string(shapeHash))
	if err != nil{
		fmt.Println("Second Delete don't", err)
	}
	// assert ink3 > ink2

	// Close the canvas.
	err = canvas.GetCanvas()
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
