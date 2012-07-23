package main

import (
	"bufio"
	"log"
	"os/exec"
	"net/smtp"
	"sync"
	"time"
)

var once sync.Once
const (
	watched = "/foo/bar"
	smtpd = "somehost:25"
	from = "you@foo"
	to = "you@bar"
	msg = "Subject: duwatcher alert. report to main bridge."
	interval = 3600
)

func main() {
	firstrun := true
	last := ""
	l := []byte("")
	for {
		cmd := exec.Command("/usr/bin/du", "-sch", watched)
		stdout,	err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		rd := bufio.NewReader(stdout)
		l, _, err = rd.ReadLine()
		cur := string(l)
		if !firstrun && cur == last {
			break
		}
		if err := cmd.Wait(); err != nil {
			log.Fatal(err)
		}
		last = cur
		once.Do(func(){
			firstrun = false
		})
		time.Sleep(interval * time.Second)
	}
	err := smtp.SendMail(smtpd, nil, from, []string{to}, []byte(msg))
	if err != nil {
		log.Fatal(err)
	}
}
