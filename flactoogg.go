package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

var artist, album, title, track string

func init() {
	flag.StringVar(&artist, "artist", "", "artist metadata")
	flag.StringVar(&album, "album", "", "album metadata")
	flag.StringVar(&title, "title", "", "title metadata")
	flag.StringVar(&track, "track", "", "track number metadata")
}

func doit(fullpath string) {
	meta(fullpath)
	//	convert(fullpath)
}

func convert(fullpath string) {
	cmd1 := exec.Command("flac", "-d", "-c", fullpath)
	stdout, err := cmd1.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd2 := exec.Command("oggenc", "-", "-o", "wat.ogg")
	stdin, err := cmd2.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd1.Start(); err != nil {
		log.Fatalf("Could not start cmd1: %v", err)
	}
	if err := cmd2.Start(); err != nil {
		log.Fatalf("Could not start cmd2: %v", err)
	}
	_, err = io.Copy(stdin, stdout)
	if err != nil {
		log.Fatalf("Could not pipe: %v", err)
	}
}

func meta(fullpath string) {
	// TODO(mpl): lookpath
	cmdname := "/usr/bin/metaflac"
	tags := make(map[string]string)
	for k, v := range map[string]string{
		"artist":      artist,
		"album":       album,
		"title":       title,
		"tracknumber": track} {
		if v == "" {
			upper := strings.ToUpper(k)
			args := []string{"--show-tag=" + upper, fullpath}
			cmd := exec.Command(cmdname, args...)
			output, err := cmd.Output()
			if err != nil {
				log.Fatalf("Could not run metaflac: %v", err)
			}
			tag := strings.Replace(string(output), upper+"=", "", 1)
			tag = strings.TrimSuffix(tag, "\n")
			tags[k] = tag
			println(tag)
			continue
		}
		tags[k] = v
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		os.Exit(1)
	}
	doit(args[0])
}
