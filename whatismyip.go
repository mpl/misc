// Copyright 2017 Mathieu Lonjaret

package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	sleepTime = time.Hour
	username = "foo"
	password = "bar"
)

func main() {
	for {
		myip, err := getip()
		if err != nil {
			log.Printf("%v", err)
			time.Sleep(sleepTime)
			continue
		}

		data := url.Values{"myip": {myip}}
		req, err := http.NewRequest("POST", "https://granivo.re:9999/upload", strings.NewReader(data.Encode()))
		if err != nil {
			log.Printf("could not prepare upload: %v", err)
			time.Sleep(sleepTime)
			continue
		}
		req.SetBasicAuth(username, password)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		dialTLS := func(network, addr string) (net.Conn, error) {
			return tls.Dial(network, addr, &tls.Config{
				InsecureSkipVerify: true,
			})
		}
		cl := &http.Client{
			Transport: &http.Transport{
				DialTLS: dialTLS,
			},
		}
		if _, err := cl.Do(req); err != nil {
			log.Printf("could not upload ip: %v", err)
			time.Sleep(sleepTime)
			continue
		}
		println("updated IP: " + myip)
		time.Sleep(sleepTime)
	}
}

func getip() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("error getting IP: %v", err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading IP: %v", err)
	}
	return string(data), nil
}
