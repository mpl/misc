/*
Copyright 2014 Mathieu Lonjaret.
*/

package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	host     = flag.String("host", "localhost:8080", "listening port and hostname")
	dir      = flag.String("dir", "~/irclogs", "dir containing the irc logs")
	chanName = flag.String("chan", "", "irc chan name")
	n        = flag.Int("n", 10, "number of lines to print")
	tlsKey   = flag.String("tlskey", "key.pem", "key for https")
	tlsCert  = flag.String("tlsCert", "cert.pem", "cert for https")
	username = flag.String("user", "", "username for HTTP basic auth")
	password = flag.String("pass", "", "password for HTTP basic auth")
	// TODO(mpl):
	verbose = flag.Bool("v", false, "verbose")
)

func main() {
	flag.Parse()
	if *chanName == "" {
		log.Fatal("Need chan")
	}
	*dir = replaceTilde(*dir)

	http.HandleFunc("/", ServeTail)
	log.Fatal(http.ListenAndServeTLS(*host, *tlsCert, *tlsKey, nil))
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

func ServeTail(w http.ResponseWriter, r *http.Request) {
	user, pass, err := basicAuth(r)
	if err != nil {
		sendUnauthorized(w, r)
		return
	}
	if user != *username || pass != *password {
		sendUnauthorized(w, r)
		return
	}

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

var kBasicAuthPattern = regexp.MustCompile(`^Basic ([a-zA-Z0-9\+/=]+)`)

func basicAuth(req *http.Request) (username, password string, err error) {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		err = fmt.Errorf("Missing \"Authorization\" in header")
		return
	}
	matches := kBasicAuthPattern.FindStringSubmatch(auth)
	if len(matches) != 2 {
		err = fmt.Errorf("Bogus Authorization header")
		return
	}
	encoded := matches[1]
	enc := base64.StdEncoding
	decBuf := make([]byte, enc.DecodedLen(len(encoded)))
	n, err := enc.Decode(decBuf, []byte(encoded))
	if err != nil {
		return
	}
	pieces := strings.SplitN(string(decBuf[0:n]), ":", 2)
	if len(pieces) != 2 {
		err = fmt.Errorf("didn't get two pieces")
		return
	}
	return pieces[0], pieces[1], nil
}

func sendUnauthorized(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("WWW-Authenticate", "Basic realm=webtail")
	rw.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(rw, "<html><body><h1>Unauthorized</h1>")
}
