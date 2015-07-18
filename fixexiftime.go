package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"camlistore.org/third_party/github.com/rwcarlsen/goexif/exif"
)

var (
	verbose = flag.Bool("v", false, "be verbose")
)

func fixTime(filename string, d time.Duration) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	ex, err := exif.Decode(f)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}	
	t, err := ex.DateTime()
	if err != nil {
		f.Close()
		log.Fatal(err)
	}
	f.Close()
	if *verbose {
		fmt.Printf("time in %v before: %v\n", filename, t)
	}
	newTime := t.Add(d).Format("2006:01:02 15:04:05")
	args := []string{
		"-t", "DateTime",
		"--set-value", newTime,
		"-o",  filename,
		"--ifd", "0",
		filename,
	}
	cmd := exec.Command("exif", args...)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	args[7] = "1"
	cmd = exec.Command("exif", args...)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	args[1] = "DateTimeOriginal"
	args[7] = "EXIF"
	cmd = exec.Command("exif", args...)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	if *verbose {
		fmt.Printf("time in %v after: %v\n", filename, newTime)
	}
}

func main() {
	flag.Parse()
	for _, v := range flag.Args() {
		fixTime(v, time.Hour)
	}
}
