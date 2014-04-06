package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/mpl/gocron"
)

var (
	emailFrom = flag.String("emailfrom", "mpl@oenone", "alert sender email address")
	notiPort  = flag.Int("port", 9688, "port for the local http server used for browser notifications")
	page      = flag.String("page", "", "page/address to scrape")
	interval  = flag.Int("interval", 3600, "Interval between runs, in seconds. use 0 to run only once.")
)

const (
	alert1 = "Subject: camlibot alert. Page not found."
	alert2 = "Subject: camlibot alert. Build or run failed."
	// TODO(mpl): regexp
	failGo1Pattern   = "/fail/linux_amd64/go1"
	failGotipPattern = "/fail/linux_amd64/gotip"
	okGo1Pattern     = "/ok/linux_amd64/go1"
	okGotipPattern   = "/ok/linux_amd64/gotip"
	datePattern      = `<td class="hash">`
	lenDate          = 19
)

var ()

var (
	latestRunTime string
	prevRunTime   string
)

func scrape() error {
	resp, err := http.Get(*page)
	if err != nil {
		return fmt.Errorf("could not fetch page at %v: %v", *page, err)
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
			return nil
		}
		prevRunTime = latestRunTime
	}

	failGo1 := bytes.Index(body, []byte(`<a href="`+failGo1Pattern))
	failGotip := bytes.Index(body, []byte(`<a href="`+failGotipPattern))
	if failGo1 == -1 && failGotip == -1 {
		return nil
	}

	goodGo1 := bytes.Index(body, []byte(`<a href="`+okGo1Pattern))
	goodGoTip := bytes.Index(body, []byte(`<a href="`+okGotipPattern))
	if (failGo1 == -1 || goodGo1 < failGo1) && (failGotip == -1 || goodGoTip < failGotip) {
		return nil
	}

	return errors.New("build or run failed.")
}

func main() {
	flag.Parse()
	if *emailFrom == "" {
		log.Fatal("Need emailfrom")
	}

	if *interval < 0 {
		log.Fatal("negative duration? what does it meeaaaann!?")
	}
	jobInterval := time.Duration(*interval) * time.Second
	cron := gocron.Cron{
		Interval: jobInterval,
		Job:      scrape,
		Mail: &gocron.MailAlert{
			Subject: "Scrape error",
			To:      []string{"mpl@mpl.fr.eu.org"},
			From:    *emailFrom,
			SMTP:    "serenity:25",
		},
		Notif: &gocron.Notification{
			Host: fmt.Sprintf("localhost:%d", *notiPort),
			Msg:  "Scrape error",
		},
		File: &gocron.StaticFile{
			Path: "/home/mpl/var/log/scrape.log",
			Msg:  "Scrape error",
		},
	}
	cron.Run()
}
