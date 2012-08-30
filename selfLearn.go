package main

import (
	"fmt"
//	"html"
	"log"
	"net/http"
)

func learnSelfHost(addr string) (c chan string) {
	c = make(chan string, 1)
	go func () {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			discoHost := r.Host
			fmt.Fprintf(w, "Hello, %q", discoHost)
			c <- discoHost
			close(c)
		})
		if err := http.ListenAndServe(addr, nil); err == nil {
			log.Fatalf("Problem during learning phase: %v", err)
		}
	}()
	return c
}

func main() {
	c := learnSelfHost("0.0.0.0:9090")
	host := <- c
	println(host)
}

