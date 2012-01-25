package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"strconv"
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("need 1 arg")
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	sum, count := float64(0), 0
	b := bufio.NewReader(f)
	for i:=0; i<3; i++ {
		b.ReadString('\n')
	}
	for {
		line, err := b.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		fields := strings.Fields(line)
		if len(fields) != 9 {
			log.Fatal("incorrect nb of fields in line")
		}
		cpu, err := strconv.ParseFloat(fields[6], 32)
		if err != nil {
			log.Fatal(err)
		}
		sum += cpu
		count++
	}
	cpu := sum / float64(count)
	log.Printf("%g \n", cpu)
}
