package main

import (
	"log"
	"os"

	"rsc.io/goversion/version"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatal("Need one argument")
	}
	v, err := version.ReadExe(args[1])
	if err != nil {
		log.Fatal(err)
	}
	println(v.ModuleInfo)
}
