package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"
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
	Tag map[string]int64
	Q300 map[string]float64
}

func (h *Hello) Search(url *string, jpl *Jpl) error {
	log.Println("received:", *url)
	jpl.Url = *url
	jpl.Tag = make(map[string]int64, 1)
	jpl.Q300 = make(map[string]float64, 1)
/*
	err := jpl.readCatdir(*url)
	if err != nil {
		return err
	}
*/
	err := jpl.query(1232476.0, 9732647.0, "foobar")
	if err != nil {
		return err
	}
	return nil
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
		jpl.Tag[species] = tag
		jpl.Q300[species] = q300
	}
	return nil
}

func (jpl *Jpl) query(fmin, fmax float64, species string) error {

	fmin_s := strconv.FormatFloat(fmin * 1e-3, 'f', 9, 64) // MHz -> GHz
	fmax_s := strconv.FormatFloat(fmax * 1e-3, 'f', 9, 64) // MHz -> GHz

	tag := "020002"
	resp, err := http.PostForm("http://spec.jpl.nasa.gov/cgi-bin/catform",
		url.Values{"MinNu": {fmin_s}, "MaxNu": {fmax_s},
		"UnitNu": {"GHz"}, "Mol": {tag}, "StrLim": {"-500"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("%v \n", string(body))
	return nil
}
