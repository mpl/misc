package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
		if err := merge(name); err != nil {
			log.Printf("%v", err)
		}
	}
}

func merge(dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	flim := ""
	sub := ""
	outFile := "out.mkv"
	for _, name := range names {
		if name == outFile {
			log.Printf("%v already contains %v, skipping this dir", dirPath, outFile)
			return nil
		}
		// TODO(mpl): take into account more extensions
		if flim == "" && strings.HasSuffix(name, ".mkv") {
			flim = name
		}
		if sub == "" && strings.HasSuffix(name, ".srt") {
			sub = name
		}
	}
	if flim == "" {
		return fmt.Errorf("no flim found in dir %v", dirPath)
	}
	if sub == "" {
		return fmt.Errorf("no sub found in dir %v", dirPath)
	}
	args := []string{
		"-o", filepath.Join(dirPath, outFile),
		filepath.Join(dirPath, flim),
		"--language", "0:fre",
		"--track-name", "0:Forced",
		"--forced-track", "0:yes",
		"--default-track", "0:yes",
		filepath.Join(dirPath, sub),
	}
	cmd := exec.Command("mkvmerge", args...)
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mkvmerge error: %v, %v", err, buf.String())
	}
	return nil
}
