package main

import (
	"crypto/sha256"
	"fmt"
)

func main() {
	// Example values
	timestamp := "1703260800000"
	method := "GET"
	path := "/trade-api/v2/portfolio/balance"

	// Construct message exactly as Kalshi expects
	msg := timestamp + method + path

	fmt.Println("Message to sign:", msg)
	fmt.Printf("Message bytes: %v\n", []byte(msg))

	// Hash it
	hashed := sha256.Sum256([]byte(msg))
	fmt.Printf("SHA256 hash: %x\n", hashed)

	// Example expected format
	fmt.Println("\nExpected message format:")
	fmt.Println("timestamp + method + path")
	fmt.Printf("Example: %s%s%s\n", timestamp, method, path)
}
