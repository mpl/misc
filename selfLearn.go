package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
)

var (
	lis     *stoppableListener
	conns   []net.Conn
	stopped = errors.New("listener stopped")
)

type signal struct{}

type stoppableListener struct {
	net.Listener
	stop chan signal
}

func (sl *stoppableListener) Accept() (c net.Conn, err error) {
	select {
	default:
		// nothing
	case <-sl.stop:
		sl.Close()
		for _, c := range conns {
			c.Close()
		}
		return nil, stopped
	}
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
		log.Fatal(err)
	}
	lis = &stoppableListener{Listener: l, stop: make(chan signal, 1)}
	learner := http.NewServeMux()
	learner.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		lis.stop <- sig
		lis.Close()
		for _, con := range conns {
			con.Close()
		}
		discoHost := r.Host
		c <- discoHost
		println("host sent")
		close(c)
	})
	go func() {
		http.Serve(lis, learner)
		println("learnSelfHost server done")
	}()
	return c
}

func createAllHandlers(host string) {
	println(host)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println("Serving with correct host")
		fmt.Fprintf(w, "Hello World")
	})
}

func main() {
	c := learnSelfHost("0.0.0.0:9090")
	host := <-c
	createAllHandlers(host)
	if err := http.ListenAndServe(host, nil); err != nil {
		log.Fatal(err)
	}
}
