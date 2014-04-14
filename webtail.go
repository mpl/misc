/*
Copyright 2014 Mathieu Lonjaret.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TODO(mpl): userpass auth

var (
	host     = flag.String("host", "localhost:8080", "listening port and hostname")
	dir      = flag.String("dir", "~/irclogs", "dir containing the irc logs")
	chanName = flag.String("chan", "", "irc chan name")
	// TODO(mpl):
	// n = flag.Int("n", 10, "number of lines to print")
	verbose = flag.Bool("v", false, "verbose")
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
	if _, err = io.Copy(w, f); err != nil {
		log.Printf("%v", err)
		http.Error(w, "nope", 500)
		return
	}
}

func main() {
	flag.Parse()
	if *chanName == "" {
		log.Fatal("Need chan")
	}
	// TODO: check ~

	http.HandleFunc("/", ServeTail)
	http.ListenAndServe(*host, nil)
}
