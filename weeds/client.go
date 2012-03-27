package main

import (
	"fmt"
	"log"
	"net/rpc/jsonrpc"
)

type Jpl struct {
	url string
	species map[int64]string
	tag map[string]int64
	q300 map[int64]float64
}

func main() {
	client, err := jsonrpc.Dial("tcp", "127.0.0.1:5090")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	// Synchronous call
	yo := "/home/mpl/work/gildas-dev/run/weeds/catdir.cat"
	args := &yo
	reply := &Jpl{}
	err = client.Call("Hello.Search", args, reply)
	if err != nil {
		log.Fatal("call error:", err)
	}
	fmt.Printf("%s", reply.url)
	println("done")
}
