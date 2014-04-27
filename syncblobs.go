package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mpl/gocron"
)

const configDir = "/home/mpl/.config/granivore/"

// TODO(mpl): prompt or server listens for password.

var (
	emailFrom = flag.String("emailfrom", "mpl@oenone", "alert sender email address")
	notiPort  = flag.Int("port", 9687, "port for the local http server used for browser notifications")
	interval  = flag.Int("interval", 3600, "Interval between runs, in seconds. use 0 to run only once.")
	auth      = flag.String("auth", "", "Use this auth string instead of the one in the config file.")
)

func syncBlobs() error {
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
	insertPos := bytes.Index(data, []byte(`"server": `))
	if insertPos < 0 {
		return noop, errors.New("insert pos not found")
	}
	authString := fmt.Sprintf("\"auth\": \"%s\",\n", *auth)
	newData := append(data[:insertPos], append([]byte(authString), data[insertPos:]...)...)
	println(string(newData))
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

func main() {
	flag.Parse()
	if *emailFrom == "" {
		log.Fatal("Need emailfrom")
	}

	if *interval < 0 {
		log.Fatal("negative duration? what does it meeaaaann!?")
	}
	if cleanup, err := fillConfig(); err != nil {
		log.Fatal(err)
	} else {
		defer cleanup()
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
