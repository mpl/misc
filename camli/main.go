package main

import (
	"bufio"
	"fmt"
//	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func camtoolSearchBlobs() {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		l := sc.Text()
		if !strings.Contains(l, `"blob":`) {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) != 2 {
			continue
		}
		fmt.Printf("%s ", strings.Replace(fields[1], `"`, "", -1))
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	camtoolSearchBlobs()
}

func whatever() {
//	var vals []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		l := sc.Text()
		if !strings.Contains(l, `"blob":`) {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) != 2 {
			continue
		}
		
		ref := strings.Replace(fields[1], `"`, "", -1)
		cmdstr := "run --rm -v /home/mpl/.config/camlistore/other:/home/camli/.config/camlistore camlistore/world camput attr -add sha1-e0d659e2da43e09470dd43919c3db16c53eba5a6 camliMember "+ref
		println(cmdstr)
		if err := exec.Command("docker", strings.Fields(cmdstr)...).Run(); err != nil {
			log.Fatal(err)
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
}
