package main

import (
	"fmt"
	"log"
	"net/rpc/jsonrpc"
//	"net/rpc"
)

type Jpl struct {
	Url string
	species map[int64]string
	Tag map[string]int64
	Q300 map[string]float64
}

func main() {
	client, err := jsonrpc.Dial("tcp", "127.0.0.1:5090")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	// Synchronous call
	yo := "/home/mpl/work/gildas-dev/run/weeds/catdir.cat"
	args := &yo
//	reply := &Jpl{}
	var reply Jpl
	err = client.Call("Hello.Search", args, &reply)
	if err != nil {
		log.Fatal("call error:", err)
	}
	fmt.Printf("%s\n", reply.Url)
	println("done")
}
