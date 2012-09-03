package main

import (
	"fmt"
	"log"
	"net/http"
	"net"
	"time"
	"sync"
)

var once sync.Once
func learnSelfHost(addr string) (c chan string) {
	c = make(chan string, 1)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	learner := http.NewServeMux()
	learner.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		once.Do(func(){
		lis.Close()
		discoHost := r.Host
		c <- discoHost
		println("host sent")
		c <- discoHost
		close(c)
		http.Redirect(w, r, "http://localhost:9090", http.StatusFound)
		})
	})
	go func () {
		if err := http.Serve(lis, learner); err == nil {
			log.Fatalf("Problem during learning phase: %v", err)
		}
		println("listener closed")
	}()
	return c
}

func main() {
	c := learnSelfHost("0.0.0.0:7070")
	host := <- c
	println(host)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you")
	})
	go http.ListenAndServe("localhost:9090", nil)
	println("main now serving")
	<- c
	time.Sleep(60 * time.Second)
}

