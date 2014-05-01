package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/mpl/gocron"
)

const configDir = "/home/mpl/.config/granivore/"

// TODO(mpl): askAuth. meh. maybe listenAuth is enough.

var (
	emailFrom  = flag.String("emailfrom", "mpl@oenone", "alert sender email address")
	notiPort   = flag.Int("port", 9687, "port for the local http server used for browser notifications")
	interval   = flag.Int("interval", 3600, "Interval between runs, in seconds. use 0 to run only once.")
	auth       = flag.String("auth", "", "Use this auth string instead of the one in the config file. Conflicts with -auth and -waitauth.")
	askAuth    = flag.Bool("askauth", false, "Prompt for the auth string on stdin. Conflicts with -auth and -waitauth.")
	listenAuth = flag.String("listenauth", "", "Listen on this address and wait for the auth string there. Conflicts with -auth and -askauth.")
)

func syncBlobs() error {
	restoreConfig, err := fillConfig()
	if err != nil {
		restoreConfig()
		return fmt.Errorf("could not prepare config: %v", err)
	}
	defer restoreConfig()
	args := []string{"sync", "-src=granivore", "-dest=/home/mpl/var/camlistore-granivore/blobs/"}
	cmd := exec.Command("/home/mpl/bin/camtool-grani", args...)
	env := os.Environ()
	env = append(env, "CAMLI_CONFIG_DIR="+configDir)
	// TODO(mpl): -verbose to see output
	// TODO(mpl): make it timeout or something in case of a 401. better yet, capture stderr,
	// and die if see anything there.
	cmd.Env = env
	return cmd.Run()
}

func fillConfig() (func() error, error) {
	noop := func() error { return nil }
	configFile := filepath.Join(configDir, "client-config.json")
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return noop, err
	}
	if *auth == "" {
		return noop, nil
	}
	erasePos, insertPos := -1, -1
	erasePos = bytes.Index(data, []byte(`"auth": `))
	if erasePos > 0 {
		insertPos = bytes.Index(data[erasePos:], []byte("\n"))
		if insertPos < 0 {
			return noop, errors.New("could not find eol for \"auth\" line")
		}
		insertPos += erasePos + 1
	} else {
		insertPos = bytes.Index(data, []byte(`"server": `))
		if insertPos < 0 {
			return noop, errors.New("insert pos not found")
		}
		erasePos = bytes.LastIndex(data[:insertPos], []byte("\n"))
		if erasePos < 0 {
			return noop, errors.New("could not find eol before \"server\" line")
		}
		erasePos++
		insertPos = erasePos
	}
	authString := fmt.Sprintf("\"auth\": \"%s\",\n", *auth)
	newData := append(data[:erasePos], append([]byte(authString), data[insertPos:]...)...)
	if err := os.Rename(configFile, configFile+".ini"); err != nil {
		return noop, err
	}
	revertConfig := func() error {
		if err := os.Rename(configFile+".ini", configFile); err != nil {
			return err
		}
		return nil
	}
	if err := ioutil.WriteFile(configFile, newData, 0700); err != nil {
		return revertConfig, err
	}
	return revertConfig, nil
}

func numSet(vv ...interface{}) (num int) {
	for _, vi := range vv {
		switch v := vi.(type) {
		case string:
			if v != "" {
				num++
			}
		case bool:
			if v {
				num++
			}
		default:
			panic("unknown type")
		}
	}
	return
}

func checkFlags() {
	if *emailFrom == "" {
		log.Fatal("Need emailfrom")
	}
	if *interval < 0 {
		log.Fatal("negative duration? what does it meeaaaann!?")
	}
	if numSet(*auth, *askAuth, *listenAuth) > 1 {
		log.Fatal("-auth, -askauth and -listenauth are mutually exclusive.")
	}
}

var authOnce sync.Once

func authHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	auth := r.FormValue("auth")
	if auth == "" {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	okSent := false
	w.WriteHeader(http.StatusOK)
	authOnce.Do(func() {
		w.Write([]byte("OK\n"))
		cAuth <- auth
		okSent = true
		// TODO(mpl): quit listening after that?
	})
	if okSent {
		return
	}
	w.Write([]byte("auth already done"))
}

var cAuth chan string

func main() {
	flag.Parse()
	checkFlags()

	if *listenAuth != "" {
		cAuth = make(chan string)
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", authHandler)
			println("Send the \"auth\" parameter in a POST request (curl -d) to " + *listenAuth)
			if err := http.ListenAndServe(*listenAuth, mux); err != nil {
				log.Fatal(err)
			}
		}()
		*auth = <-cAuth
	}

	jobInterval := time.Duration(*interval) * time.Second
	cron := gocron.Cron{
		Interval: jobInterval,
		Job:      syncBlobs,
		Mail: &gocron.MailAlert{
			Subject: "Syncblobs error",
			To:      []string{"mpl@mpl.fr.eu.org"},
			From:    *emailFrom,
			SMTP:    "serenity:25",
		},
		Notif: &gocron.Notification{
			Host: fmt.Sprintf("localhost:%d", *notiPort),
			Msg:  "Syncblobs error",
		},
		File: &gocron.StaticFile{
			Path: "/home/mpl/var/log/syncblobs.log",
			Msg:  "gocron error",
		},
	}
	cron.Run()
}
