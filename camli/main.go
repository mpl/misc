package main

import (
	"bufio"
	"bytes"
	"fmt"
//	"io"
	"log"
	"math/rand"
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
	manyPermanodesWithLocation(1000)
}

func idontremember() {
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

func permanodeWithLocation(lat, long float64) error {
	cmd := exec.Command("devcam", "put", "permanode", "-tag", "fake")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	sc := bufio.NewScanner(bytes.NewReader(out))
	var pn string
	for sc.Scan() {
		l := sc.Text()
		if !strings.HasPrefix(l, "sha1-") {
			return fmt.Errorf("unexpected first line of output: %v", l)
		}
		pn = strings.TrimSpace(l)
		break
	}
	
	cmd = exec.Command("devcam", "put", "attr", pn, "latitude", fmt.Sprintf("%f", lat))
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("devcam", "put", "attr", pn, "longitude", fmt.Sprintf("%f", long))
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func manyPermanodesWithLocation(limit int) {
	for i:=0; i<limit; i++ {
		lat := rand.Float64()*89.99 - rand.Float64()*89.99
		long := rand.Float64()*179.99 - rand.Float64()*179.99
		if err := permanodeWithLocation(lat, long); err != nil {
			log.Fatal(err)
		}
	}
}
