package main

import (
	//	"bufio"
	//	"bytes"
	//	"errors"
	"flag"
	//	"fmt"
	//	"io/ioutil"
	"log"
	"os"
	"os/exec"
	//	"path/filepath"
	"strings"
	"time"

	//	"github.com/mpl/gocron"
)

var (
	emailFrom = flag.String("emailfrom", "mpl@serenity", "alert sender email address")
	interval  = flag.Int("interval", 60, "Interval between runs, in seconds. use 0 to run only once.")
	bin       = flag.String("binPath", "/home/mpl/gocode/bin/rtorrentrpc", "path to the rtorrentrpc binary to use.")
)

var (
	currentBinding string
	ipredIP        string
	retryPause     = 1 * time.Second
)

func getBinding() ([]byte, error) {
	args := []string{"localhost:5000", "get_bind"}
	cmd := exec.Command(*bin, args...)
	cmd.Env = os.Environ()
	return cmd.Output()
}

// TODO(mpl): do it with regexp

const (
	posHint = "<param><value><string>"
	endHint = "</string></value></param>"
)

func getIP(xml string) string {
	idx := strings.Index(xml, "<param><value><string>")
	if idx <= 0 {
		println("no beg pos")
		return ""
	}
	begin := idx + len(posHint)
	xml = xml[begin:]
	idx = strings.Index(xml, endHint)
	if idx <= 0 {
		println("no end pos")
		return ""
	}
	return xml[:idx]
}

func setBinding() ([]byte, error) {
	args := []string{"localhost:5000", "set_bind", ipredIP}
	cmd := exec.Command(*bin, args...)
	cmd.Env = os.Environ()
	return cmd.Output()
}

func checkBinding() error {
	for {
		println("looping")
		time.Sleep(retryPause)
		xml, err := getBinding()
		println(string(xml))
		if err != nil {
			continue
		}
		xmlString := string(xml)
		if xmlString == "" {
			continue
		}
		currentBinding = getIP(xmlString)
		if currentBinding == "" {
			continue
		}
		println(currentBinding)
		if currentBinding == ipredIP {
			println("ALL GOOD")
			return nil
		}
		for {
			time.Sleep(retryPause)
			xml, err := setBinding()
			if err != nil {
				continue
			}
			xmlString := string(xml)
			if xmlString == "" {
				continue
			}
			break
		}
	}
	return nil
}

func checkFlags() {
	if *emailFrom == "" {
		log.Fatal("Need emailfrom")
	}
	if *bin == "" {
		log.Fatal("Need binPath")
	}
	if *interval < 0 {
		log.Fatal("negative duration? what does it meeaaaann!?")
	}
	if len(flag.Args()) != 1 {
		log.Fatal("need current ipred ip as argument")
	}
}

func main() {
	flag.Parse()
	checkFlags()
	ipredIP = flag.Args()[0]
	checkBinding()

	/*
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
	*/
}
