package main

import (
	"fmt"
  	"crypto/elliptic"
  	"crypto/rand"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"os"
)


func main() {
	for i := 0; i < 20; i++ {
		public, private := GetPublicPrivateKeyPair()
		fmt.Println("public", public)
		fmt.Println("private", private)
		fmt.Println("")
	}
}


func GetPublicPrivateKeyPair() (string, string) {
	priv1, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	exitOnError("ink-miner GetPublicPrivateKeyPair", err)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&priv1.PublicKey)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(priv1)
	encodedPublicBytes := hex.EncodeToString(publicKeyBytes)
	encodedPrivateBytes := hex.EncodeToString(privateKeyBytes)

	return encodedPublicBytes, encodedPrivateBytes
}

func exitOnError(prefix string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s, err = %s\n", prefix, err.Error())
		os.Exit(1)
	}
}