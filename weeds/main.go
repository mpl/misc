package main

import (
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
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

type Searcher interface {
	Search(species string, fmin float, fmax float)
}

type Jpl struct {
	url string
	species map[int64]string
	tag map[string]string
	q300 map[int64]float64
}

func (h *Hello) Search(url *string, jpl *Jpl) error {
	log.Println("received:", *url)

}

func Query(string species, fmin, fmax float) {

}

func (jpl *Jpl) readCatdir(path string) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		l, _, err := f.ReadLine()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		tag, err := strconv.ParseInt(string(l[0:6]), 0, 0)
		if err != nil {
			return err
		}
		species := strings.TrimSpace(l[7:20])
		exp, err := strconv.ParseFloat(stringl[26:33], 0, 0)
		if err != nil {
			return err
		}
		q300 := math.Pow(10, exp)
		jpl.species[tag] = species
		jpl.tag[species] = tag
		jpl.q300[tag] = q300
	}
}
