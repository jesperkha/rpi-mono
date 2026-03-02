package main

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hashpassword <password>")
		os.Exit(1)
	}

	password := os.Args[1]
	hash := sha256.Sum256([]byte(password))
	fmt.Printf("%x\n", hash)
}
