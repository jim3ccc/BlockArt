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
//import "time"
//import "crypto/ecdsa"

func main() {
    minerAddr := "127.0.0.1:56319"
    privKey := "3081dc020101044201c58b110afbd060dd326cef4600967efa19c44fe2a492ea734d86d16ec9367f6f243f2b0d5a81903bf12008f20fc97b1b6b6addd406d6eda68a6a3cdcf8b65a8fc2a00706052b81040023a18189038186000400ce50f0de65becd4bb2e06f48939e541ef4f397f3c078afee15f85343f1bb70e41376b70b41d5eee43c9cfdb3d869e24a48f0835dc4825eaa922efabd4d6e9c2aa00182099b0e9263bfa1bd4912e02f47ee878bba4c92a9c7a3d00a149a1819988323d68002f5361d2d2bab56a25443c29810daf60ca3c72a4014e7ac616f59b527d36b"
    privKeyBytesRestored, _ := hex.DecodeString(privKey)
    private_key, _ := x509.ParseECPrivateKey(privKeyBytesRestored)
    // Open a canvas.
    canvas, _, err := blockartlib.OpenCanvas(minerAddr, *private_key)
    if err != nil{
        fmt.Println("0", err)
    }

    validateNum := uint8(2)

    // Add a line.
    shapeHash, _, _, err := canvas.AddShape(validateNum, shared.PATH, "M 50 50 L 0 5", "transparent", "red")
    if err != nil{
        fmt.Println("3", err)
    }

    // Add another line.
    _, _, _, err = canvas.AddShape(validateNum, shared.PATH, "M 0 0 h 200 v 200 l -50 -150 z", "black", "blue")
    if err != nil{
        fmt.Println("4", err)
    }

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
