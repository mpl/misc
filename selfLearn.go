package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net"
	"time"
	"sync"
)

var once sync.Once
var lis *stoppableListener
var conns []net.Conn
var stopped = errors.New("listener stopped")

type signal struct{}

type stoppableListener struct {
        net.Listener
        stop chan signal
}

func (sl *stoppableListener) Accept() (c net.Conn, err error) {
		// non-blocking read on the stop channel
		select {
		default:
				// nothing
		case <-sl.stop:
				for _,c := range conns {
					c.Close()
				}
				return nil, stopped
		}

		// if we got here, we have not been asked to stop, so call
		// Accept on the underlying listener.

		c, err = sl.Listener.Accept()
		if err != nil {
				return
		}
		conns = append(conns, c)
		return
}

func learnSelfHost(addr string) (c chan string) {
	c = make(chan string, 1)
	var sig signal
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	lis = &stoppableListener{Listener: l, stop: make(chan signal, 1)}
	learner := http.NewServeMux()
	learner.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println("been there")
		once.Do(func(){
			lis.stop <- sig
			println("done that")
			discoHost := r.Host
			c <- discoHost
			println("host sent")
			c <- discoHost
			close(c)
/*
			w.Header().Set("Connection", "close")
			hj, ok := w.(http.Hijacker)
			if !ok {
				log.Fatalf("no hj")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				log.Fatal(err)
			}
			conn.Close()
*/
//			http.Redirect(w, r, "http://localhost:9090", http.StatusFound)
		})
	})
	go func () {
		if err := http.Serve(lis, learner); err == nil {
			log.Fatalf("Problem during learning phase: %v", err)
		}
		println("server done")
	}()
	return c
}

func main() {
	c := learnSelfHost("0.0.0.0:9090")
	host := <- c
	println(host)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you")
	})
	go func() {
		if err := http.ListenAndServe("localhost:9090", nil); err != nil {
			log.Fatal(err)
		}
	}()
	println("main now serving")
	<- c
	time.Sleep(60 * time.Second)
}

