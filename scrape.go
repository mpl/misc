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
	// TODO(mpl): regexp
	failGo1Pattern   = "/fail/linux_amd64/go1"
	failGotipPattern = "/fail/linux_amd64/gotip"
	okGo1Pattern     = "/ok/linux_amd64/go1"
	okGotipPattern   = "/ok/linux_amd64/gotip"
	datePattern      = `<td class="hash">`
	lenDate          = 19
)

var (
	page      = flag.String("page", "", "page/address to scrape")
	emailTo   = flag.String("emailto", "", "address where to send an alert when failing")
	emailFrom = flag.String("emailfrom", "", "alert sender email address")
	smtpAddr  = flag.String("smtp", "localhost:25", "where to relay the message")
)

var (
	latestRunTime string
	prevRunTime   string
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

	datePos := bytes.Index(body, []byte(datePattern)) + len(datePattern)
	latestRunTime = string(body[datePos : datePos+lenDate])
	if prevRunTime == "" {
		prevRunTime = latestRunTime
	} else {
		// TODO(mpl): actually parse them as Time and properly compare.
		// whatever. I have the flu so I'm allowed.
		if latestRunTime == prevRunTime {
			return
		}
		prevRunTime = latestRunTime
	}

	failGo1 := bytes.Index(body, []byte(`<a href="`+failGo1Pattern))
	failGotip := bytes.Index(body, []byte(`<a href="`+failGotipPattern))
	if failGo1 == -1 && failGotip == -1 {
		return
	}

	goodGo1 := bytes.Index(body, []byte(`<a href="`+okGo1Pattern))
	goodGoTip := bytes.Index(body, []byte(`<a href="`+okGotipPattern))
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
