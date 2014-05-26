package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"
)

const (
	marker  = "Kom"
	logFile = "errors.log"
)

var (
	dryrun = flag.Bool("dryrun", false, "print action, but don't actually do the renaming")
)

func main() {
	flag.Parse()
	logger, err := os.Create(logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			panic(err)
		}
	}()
	log.SetOutput(logger)

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
		if !strings.HasSuffix(name, ".doc") {
			continue
		}
		kom, err := grep(name, marker)
		if err != nil {
			log.Printf("could not grep kom number in %v: %v", name, err)
			continue
		}
		log.Printf("found kom number: %v", kom)
		kom = strings.Replace(kom, " ", "", -1)
		kom = strings.Replace(kom, "=", "", -1)
		kom = strings.Replace(kom, "/", "_", -1)
		kom = strings.Replace(kom, ":", "-", -1)
		newName := kom + ".doc"
		if newName == name {
			continue
		}
		if *dryrun {
			log.Printf("would rename %v into %v", name, newName)
			continue
		}
		if _, err := os.Stat(newName); err == nil {
			log.Printf("renaming %v into %v would overwrite an existing file. won't do it.", name, newName)
			continue
		}
		if err := os.Rename(name, newName); err != nil {
			log.Printf("error renaming %v into %v: %v", name, newName, err)
			continue
		}
	}
}

func grep(filePath string, marker string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Split(scanMSWordLines)
	for sc.Scan() {
		line := sc.Text()
		markerPos := strings.Index(line, marker)
		if markerPos < 0 {
			continue
		}
		komPos := markerPos
		for pos, r := range line[markerPos:] {
			if unicode.IsDigit(r) {
				komPos += pos
				break
			}
		}
		if komPos == markerPos {
			log.Printf("could not find beginning of actual Kom number after %q marker", marker)
			continue
		}
		return strings.TrimSpace(line[komPos:]), nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("could not find marker %q in %v", marker, filePath)
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

// scanMSWordLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return.
// The last non-empty line of input will be returned even if it has no
// newline.
func scanMSWordLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}
