package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/mpl/gocron"
)

var (
	emailFrom = flag.String("emailfrom", "mpl@oenone", "alert sender email address")
	notiPort  = flag.Int("port", 9687, "port for the local http server used for browser notifications")
	once      = flag.Bool("once", false, "whether to run this sync only once instead of every hour")
)

func syncBlobs() error {
	args := []string{"sync", "-src=granivore", "-dest=/home/mpl/var/camlistore-granivore/blobs/"}
	cmd := exec.Command("/home/mpl/bin/camtool-grani", args...)
	env := os.Environ()
	env = append(env, "CAMLI_CONFIG_DIR=/home/mpl/.config/granivore/")
	// TODO(mpl): -verbose to see output
	// TODO(mpl): make it timeout or something in case of a 401. better yet, capture stderr,
	// and die if see anything there.
	cmd.Env = env
	return cmd.Run()
}

func main() {
	flag.Parse()
	if *emailFrom == "" {
		log.Fatal("Need emailfrom")
	}

	interval := time.Hour
	if *once {
		interval = 0
	}
	cron := gocron.Cron{
		Interval: interval,
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
