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
		go jsonrpc.ServeConn(c)
	}
}

type Jpl struct {
	url string
	species map[int64]string
	tag map[string]int64
	q300 map[int64]float64
}

func (h *Hello) Search(url *string, jpl *Jpl) error {
	log.Println("received:", *url)
	jpl = &Jpl{
		url: *url,
		species: make(map[int64]string, 1),
		tag: make(map[string]int64, 1),
		q300: make(map[int64]float64, 1),
	}
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
		jpl.tag[species] = tag
		jpl.q300[tag] = q300
	}
	return nil
}
