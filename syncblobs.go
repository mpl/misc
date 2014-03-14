package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"net/smtp"
	"time"
)

const (
	alert   = "Subject: syncBlobs alert."
	interval = time.Hour
)

var (
	emailTo   = flag.String("emailto", "mpl@smgl.fr.eu.org", "address where to send an alert when failing")
	emailFrom = flag.String("emailfrom", "mpl@oenone", "alert sender email address")
	smtpAddr  = flag.String("smtp", "serenity:25", "where to relay the message")
)

func syncBlobs() {
	args := []string{"sync", "-src=granivore", "-dest=/home/mpl/var/camlistore-granivore/blobs/"}
	cmd := exec.Command("/home/mpl/bin/camtool-grani", args...)
	env := os.Environ()
	env = append(env, "CAMLI_CONFIG_DIR=/home/mpl/.config/granivore/")
	// TODO(mpl): -verbose to see output
	// TODO(mpl): make it timeout or something in case of a 401. better yet, capture stderr,
	// and die if see anything there.
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		msg := fmt.Sprintf("%s\n\n%v", alert, err)
		err = SendMail(*smtpAddr, *emailFrom, []string{*emailTo}, []byte(msg))
		if err != nil {
			// TODO(mpl): show notification :)
			log.Fatal(err)
		}
	}
}

// SendMail connects to the server at addr, authenticates with the
// optional mechanism a if possible, and then sends an email from
// address from, to addresses to, with message msg.
func SendMail(addr string, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

func main() {
	flag.Parse()
	if *emailTo == "" || *emailFrom == "" {
		log.Fatal("Need emailfrom and emailto")
	}
	// TODO(mpl): Y U NO SHOW FROM anymore?
	for {
		syncBlobs()
		time.Sleep(interval)
	}
}
