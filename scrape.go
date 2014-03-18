package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"time"
)

const (
	alert1   = "Subject: camlibot alert. Page not found."
	alert2   = "Subject: camlibot alert. Build or run failed."
	interval = time.Hour
)

var (
	page      = flag.String("page", "", "page/address to scrape")
	emailTo   = flag.String("emailto", "", "address where to send an alert when failing")
	emailFrom = flag.String("emailfrom", "", "alert sender email address")
	smtpAddr  = flag.String("smtp", "localhost:25", "where to relay the message")
)

func scrape() {
	resp, err := http.Get(*page)
	if err != nil {
		err = SendMail(*smtpAddr, *emailFrom, []string{*emailTo}, []byte(alert1))
		if err != nil {
			log.Printf("%v", err)
		}
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	failGo1 := bytes.Index(body, []byte(`<a href="/fail/go1`))
	failGotip := bytes.Index(body, []byte(`<a href="/fail/gotip`))
	if failGo1 == -1 && failGotip == -1 {
		return
	}
	goodGo1 := bytes.Index(body, []byte(`<a href="/ok/go1`))
	goodGoTip := bytes.Index(body, []byte(`<a href="/ok/gotip`))
	if (failGo1 == -1 || goodGo1 < failGo1) && (failGotip == -1 || goodGoTip < failGotip) {
		return
	}
	err = SendMail(*smtpAddr, *emailFrom, []string{*emailTo}, []byte(alert2))
	if err != nil {
		log.Printf("%v", err)
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
	if *page == "" {
		log.Fatal("Need a page to scrape")
	}
	if *emailTo == "" || *emailFrom == "" {
		log.Fatal("Need emailfrom and emailto")
	}
	// TODO(mpl): Y U NO SHOW FROM anymore?
	for {
		scrape()
		time.Sleep(interval)
	}
}
