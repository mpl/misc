package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mpl/gocron"
)

var (
	emailFrom = flag.String("emailfrom", "", "alert sender email address")
	notiPort  = flag.Int("port", 0, "port for the local http server used for browser notifications")
	page      = flag.String("page", "", "page/address to scrape")
	interval  = flag.Int("interval", 3600, "Interval between runs, in seconds. use 0 to run only once.")
	test      = flag.Bool("test", false, "Notify of everything, not just failures.")
	verbose   = flag.Bool("v", false, "verbose")
)

const (
	// TODO(mpl): regexp
	failGo1Pattern   = "/fail/linux_amd64/go1"
	failGotipPattern = "/fail/linux_amd64/gotip"
	okGo1Pattern     = "/ok/linux_amd64/go1"
	okGotipPattern   = "/ok/linux_amd64/gotip"
	datePattern      = `<td class="hash">`
	lenDate          = 19
)

var (
	latestRunTime string
	prevRunTime   string
)

func getPage() ([]byte, error) {
	if !strings.HasPrefix(*page, "http://") {
		// assume local file. for testing.
		return ioutil.ReadFile(*page)
	}
	resp, err := http.Get(*page)
	if err != nil {
		return nil, fmt.Errorf("could not fetch page at %v: %v", *page, err)
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func scrape() error {
	body, err := getPage()
	if err != nil {
		// TODO(mpl): logger exists only when *verbose
		if *verbose {
			log.Printf("%v", err)
		}
		return err
	}

	datePos := bytes.Index(body, []byte(datePattern)) + len(datePattern)
	latestRunTime = string(body[datePos : datePos+lenDate])
	if prevRunTime == "" {
		prevRunTime = latestRunTime
	} else {
		// TODO(mpl): actually parse them as Time and properly compare.
		// whatever. I have the flu so I'm allowed.
		if latestRunTime == prevRunTime {
			if *verbose {
				log.Print("No new run")
			}
			if *test {
				return fmt.Errorf("No new run")
			}
			return nil
		}
		prevRunTime = latestRunTime
	}

	failGo1 := bytes.Index(body, []byte(`<a href="`+failGo1Pattern))
	failGotip := bytes.Index(body, []byte(`<a href="`+failGotipPattern))
	if failGo1 == -1 && failGotip == -1 {
		if *verbose {
			log.Print("No fail at all")
		}
		if *test {
			return fmt.Errorf("No fail at all")
		}
		return nil
	}

	goodGo1 := bytes.Index(body, []byte(`<a href="`+okGo1Pattern))
	goodGoTip := bytes.Index(body, []byte(`<a href="`+okGotipPattern))
	if (failGo1 == -1 || goodGo1 < failGo1) && (failGotip == -1 || goodGoTip < failGotip) {
		if *verbose {
			log.Print("No recent fail")
		}
		if *test {
			return fmt.Errorf("No recent fail")
		}
		return nil
	}

	log.Print("build or run failed.")
	return errors.New("build or run failed.")
}

func main() {
	flag.Parse()
	var mailAlert *gocron.MailAlert
	if *emailFrom != "" {
		mailAlert = &gocron.MailAlert{
			Subject: "Scrape error",
			To:      []string{"mpl@mpl.fr.eu.org"},
			From:    *emailFrom,
			SMTP:    "serenity:25",
		}
	}

	if *interval < 0 {
		log.Fatal("negative duration? what does it meeaaaann!?")
	}
	jobInterval := time.Duration(*interval) * time.Second
	cron := gocron.Cron{
		Interval: jobInterval,
		Job:      scrape,
		Mail:     mailAlert,
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
