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
// import "time"
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
	}

	shapeHash, _, _, err = canvas.AddShape(validateNum, shared.PATH, "M 543 479 L 280 58 L 328 480", "transparent", "red")
	if err != nil{
		fmt.Println("trans red", err)
	}

	shapeHash, _, _, err = canvas.AddShape(validateNum, shared.PATH, "M 162 0 L 0 32 L 0 120", "transparent", "yellow")
	if err != nil{
		fmt.Println("blue yellow", err)
	}


	shapeHash, _, _, err = canvas.AddShape(validateNum, shared.PATH, "M 0 6 L 0 5", "transparent", "HaiLel")
	if err != nil{
		fmt.Println("5", err)
	}
	// Add another line.
	_, _, _, err = canvas.AddShape(validateNum, shared.PATH, "M 0 0 h 200 v 200 l -50 -150 z", "transparent", "blue")
	if err != nil{
		fmt.Println("6", err)
	}
	// time.Sleep(20000 * time.Millisecond)

	err = canvas.GetCanvas()
	if err != nil {
		fmt.Println("8", err)

	}


	// Delete the first line.
	_, err = canvas.DeleteShape(validateNum, shapeHash)
	if err != nil{
		fmt.Println("6", err)
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
