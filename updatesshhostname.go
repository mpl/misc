/*
Copyright 2015 Mathieu Lonjaret

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	flagConfigFile = flag.String("config", "", "ssh config file. Defaults to ~/.ssh/config.")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: updatesshhostname HostPattern NewHostname|NewIP\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func backup(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(filename+".0", data, 0600); err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		usage()
	}
	host := args[0]
	hostname := args[1]
	if *flagConfigFile == "" {
		*flagConfigFile = filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	}
	f, err := os.Open(*flagConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var buf bytes.Buffer
	hostMarker := "Host " + host
	hostnameMarker := "	Hostname	"
	found := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if _, err := buf.WriteString(l + "\n"); err != nil {
			log.Fatal(err)
		}
		if l != hostMarker {
			continue
		}
		for sc.Scan() {
			l := sc.Text()
			if l == "\n" {
				log.Fatalf("Could not find hostname in host section of %v", host)
			}
			if strings.HasPrefix(l, hostnameMarker) {
				found = true
				l = hostnameMarker + hostname
			}
			if _, err := buf.WriteString(l + "\n"); err != nil {
				log.Fatal(err)
			}
			if found {
				break
			}
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	if !found {
		log.Fatal("No such host found")
	}
	backup(*flagConfigFile)
	if err := ioutil.WriteFile(*flagConfigFile, buf.Bytes(), 0600); err != nil {
		log.Fatal(err)
	}
}
