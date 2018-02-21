package miners

import (
	"../shared"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
)

func getSecretFromBlock(block *shared.Block) string {
	secret := ""
	secret += block.Parent
	for _, op := range block.Operations {
		secret += op.Op
		secret += op.OpSig
	}
	secret += shared.PubKeyToString(block.Creator)

	return secret
}

func computeSecretNumber(nonce string, secret int, n int, stop chan bool) string {
	for {
		select {
		case forceStop := <-stop:
      if forceStop {
  			// plz stop as instructed
  			fmt.Println("Stopped block while mining!")
  			return ""
      }
		default:
			hash := computeNonceSecretHash(nonce, strconv.Itoa(secret))
			// if matcher.MatchString(hash) {
			if CheckLastNOfHash(n, hash) {
				// fmt.Printf("The secret is %v!\n", strconv.Itoa(secret))
				// fmt.Printf("hash is %s \n", hash)
				return strconv.Itoa(secret)
			}
			secret += 1
		}
	}
}

func CheckLastNOfHash(n int, hash string) bool {
	valid := true
	for i := len(hash) - 1; i >= len(hash)-n; i -= 1 {
		if hash[i] != 48 {
			valid = false
		}
	}
	return valid
}

func BruteForceNonce(block *shared.Block, n int64, stop chan bool) {
	secret := getSecretFromBlock(block)

	block.Nonce = computeSecretNumber(secret, 1, int(n), stop)

	return
}

// Returns the MD5 hash as a hex string for the  (secret + nonce) value.
func computeNonceSecretHash(nonce string, secret string) string {
	h := md5.New()
	h.Write([]byte(nonce + secret))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}

func ComputeHash(block shared.Block) string {
	return computeNonceSecretHash(getSecretFromBlock(&block), block.Nonce)
}
