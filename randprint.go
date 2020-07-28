package main

import (
	"crypto/rand"
	"fmt"
)

func myrand(size int) string {
	buf := make([]byte, size)
	if n, err := rand.Read(buf); err != nil || n != len(buf) {
		panic("failed to get random: " + err.Error())
	}
	return fmt.Sprintf("%x", buf)
}

func main() {
	fmt.Printf(`println("%s")`, myrand(10))
}
