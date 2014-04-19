/*
Copyright 2014 Mathieu Lonjaret.
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TODO(mpl): userpass auth

var (
	host     = flag.String("host", "localhost:8080", "listening port and hostname")
	dir      = flag.String("dir", "~/irclogs", "dir containing the irc logs")
	chanName = flag.String("chan", "", "irc chan name")
	n        = flag.Int("n", 10, "number of lines to print")
	// TODO(mpl):
	verbose = flag.Bool("v", false, "verbose")
)

var (
	lazyMu         sync.Mutex
	currentLogFile string
	currentModTime time.Time
	currentLines   []string
)

func latestLogFile() (string, error) {
	d, err := os.Open(*dir)
	if err != nil {
		return "", err
	}
	defer d.Close()
	fi, err := d.Stat()
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("%v not a dir", *dir)
	}
	fis, err := d.Readdir(-1)
	if err != nil {
		return "", err
	}
	chanSuffix := "#" + *chanName + ".log"
	latestLogFile := ""
	latest := time.Time{}
	for _, v := range fis {
		if !strings.HasSuffix(v.Name(), chanSuffix) {
			continue
		}
		if v.ModTime().After(latest) {
			latest = v.ModTime()
			latestLogFile = v.Name()
		}
	}
	return latestLogFile, nil
}

func getLines(f io.Reader) ([]string, error) {
	lines := make([]string, *n)
	index := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines[index] = scanner.Text()
		if index == *n-1 {
			index = 0
		} else {
			index++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if index != 0 {
		lines = append(lines[index:*n], lines[0:index]...)
	}
	return lines, nil
}

func ServeTail(w http.ResponseWriter, r *http.Request) {
	logFile, err := latestLogFile()
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "file not found", 404)
		return
	}
	f, err := os.Open(filepath.Join(*dir, logFile))
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "file not found", 404)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		log.Printf("could not stat %v: %v", logFile, err)
		http.Error(w, "could not stat", 500)
		return
	}
	// TODO(mpl): be nicer with the lock. getLines shouldn't be in the locked zone.
	lazyMu.Lock()
	if logFile != currentLogFile || fi.ModTime().After(currentModTime) {
		lines, err := getLines(f)
		if err != nil {
			log.Printf("could not get lines from %v: %v", logFile, err)
			http.Error(w, "nope", 500)
			return
		}
		currentLines = lines
		currentModTime = fi.ModTime()
		currentLogFile = logFile
	}
	for _, v := range currentLines {
		fmt.Fprintf(w, "%s\n", v)
	}
	lazyMu.Unlock()
	return
}

func replaceTilde(filePath string) string {
	if !strings.Contains(filePath, "~") {
		return filePath
	}
	e := os.Getenv("HOME")
	if e == "" {
		log.Fatal("~ in file path but $HOME not defined")
	}
	return strings.Replace(filePath, "~", e, -1)
}

func main() {
	flag.Parse()
	if *chanName == "" {
		log.Fatal("Need chan")
	}
	*dir = replaceTilde(*dir)

	http.HandleFunc("/", ServeTail)
	http.ListenAndServe(*host, nil)
}
