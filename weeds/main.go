package main

import (
	"bufio"
	"io"
	"log"
	"math"
	"os"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"
)

type Hello int

func (h *Hello) Hello(arg *string, reply *string) error {
	log.Println("received:", *arg)
	*reply = "hello"
	return nil
}

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:5090")
	defer l.Close()

	if err != nil {
		log.Fatal(err)
	}

	log.Print("listening:", l.Addr())

	hello := new(Hello)
	rpc.Register(hello)

	for {
		log.Print("waiting for connections...")
		c, err := l.Accept()

		if err != nil {
			log.Printf("accept error: %s", c)
			continue
		}

		log.Printf("connection started: %v", c.RemoteAddr())
//		go rpc.ServeConn(c)
		go jsonrpc.ServeConn(c)
	}
}

type Jpl struct {
	Url string
	species map[int64]string
	Tag map[string]int64
	Q300 map[string]float64
}

func (h *Hello) Search(url *string, jpl *Jpl) error {
	log.Println("received:", *url)
//	*foo = *url
	jpl.Url = *url
	jpl.species = make(map[int64]string, 1)
	jpl.Tag = make(map[string]int64, 1)
	jpl.Q300 = make(map[string]float64, 1)
	return jpl.readCatdir(*url)
}

func (jpl *Jpl) readCatdir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bf := bufio.NewReader(f)
	for {
		l, _, err := bf.ReadLine()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		tag, err := strconv.ParseInt(strings.TrimSpace(string(l[0:6])), 0, 0)
		if err != nil {
			return err
		}
		species := strings.TrimSpace(string(l[7:20]))
		exp, err := strconv.ParseFloat(strings.TrimSpace(string(l[26:33])), 64)
		if err != nil {
			return err
		}
		q300 := math.Pow(10, exp)
		jpl.species[tag] = species
		jpl.Tag[species] = tag
		jpl.Q300[species] = q300
	}
	return nil
}
