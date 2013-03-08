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
	smtpd    = "localhost:25"
	alert1   = "Subject: camlibot alert. Page not found."
	alert2   = "Subject: camlibot alert. Build or run failed."
	interval = time.Hour
)

var (
	page      = flag.String("page", "", "page/address to scrape")
	emailTo   = flag.String("emailto", "", "address where to send an alert when failing")
	emailFrom = flag.String("emailfrom", "", "alert sender email address")
)

func scrape() {
	resp, err := http.Get(*page)
	if err != nil {
		err = smtp.SendMail(smtpd, nil, *emailFrom, []string{*emailTo}, []byte(alert1))
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fail := bytes.Index(body, []byte(`<a href="/fail`))
	if fail == -1 {
		return
	}
	good := bytes.Index(body, []byte(`<a href="/ok`))
	if good < fail {
		return
	}
	err = smtp.SendMail(smtpd, nil, *emailFrom, []string{*emailTo}, []byte(alert2))
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()
	if *page == "" {
		log.Fatal("Need a page to scrape")
	}
	if *emailTo == "" || *emailFrom == "" {
		log.Fatal("Need emailfrom and emailto")
	}
	for {
		scrape()
		time.Sleep(interval)
	}
}
