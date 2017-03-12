package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

var (
	flagDoit = flag.Bool("doit", false, "actually do the renaming. otherwise just print what would happen.")
)

var reg = regexp.MustCompile(`(^.*-mp3.zip).*`)

func main() {
	flag.Parse()

	dir, err := os.Open(".")
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	for _, name := range names {
		if !strings.Contains(name, "mp3.zip?gamekey") {
			continue
		}
		m := reg.FindStringSubmatch(name)
		if len(m) != 2 {
			continue
		}
		newname := m[1]
		if *flagDoit {
			if err := os.Rename(name, newname); err != nil {
				log.Fatalf("could not rename %q to %q: %v", name, newname, err)
			}
			continue
		}
		fmt.Printf("%q -> %q\n", name, newname)
	}
}
