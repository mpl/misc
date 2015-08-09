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
	webDestPort = flag.String("webport", "8080", "port that will get all packets destined to port 80")
	webDestPortTLS = flag.String("webportTLS", "4443", "port that will get all packets destined to port 443")
)

var (
	currentBinding string
	ipredIP        string
	boundIP string
	retryPause     = 1 * time.Second
)

const tun = "tun100"
const noTunMsg = tun +": error fetching interface information: Device not found"
var noTunErr = errors.New(noTunMsg)
var noRtorrentErr = errors.New("rtorrent not running")


func getTunIP() (string, error) {
	// TODO(mpl): can probably be done with the stdlib.
	cmd, err := exec.Command("/sbin/ifconfig", tun)
	if err != nil {
		return "", err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), noTunMsg) {
			return "", noTunErr
		}
		return "", fmt.Errorf("%v: %v", err, string(out))
	}
	sc, err := bufio.Scanner()
	if err != nil {
		return "", err
	}
	for sc.Scan() {
		l := sc.Text()
		if !strings.HasPrefix(l, "inet addr:") {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) != 3 {
			return "", fmt.Errorf("wrong number of parts in inet addr line")
		}
		return strings.TrimSpace(strings.TrimPrefix(parts[0], "inet addr:")), nil
	}
}

func runOrDie(cmd, args...) {
	cmd, err := exec.Command(cmd, args...)
	if err != nil {
		// TODO(mpl): warn me
		log.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "" {
		// TODO(mpl): warn me
		log.Fatalf("%v: %v", err, string(out))
	}	
}
	
func main() {
	for {
		ip, err := getTunIP()
		if err != nil {
			if err != noTunErr {
				// TODO(mpl): warn me
				log.Fatal(err)
			}
			// TODO(mpl): restart openvpn, and loop back ?
		}
		if ip == ipredIP {
			// TODO(mpl): maybe not, if we're here because last rtorrent reset failed.
			time.Sleep(*interval)
			continue
		}
		ipredIP = ip

		// mark packets that should go through the tunnel
		runOrDie(strings.Split("iptables -t nat -F", " ")...)
		runOrDie(strings.Split("iptables -t mangle -F", " ")...)
		runOrDie(strings.Split("iptables -t mangle -A OUTPUT --source "+ip+" -j MARK --set-mark 1", " ")...)
		runOrDie(strings.Split("iptables -t nat -A POSTROUTING -o "+tun+" -j SNAT --to "+ip, " ")...)
		runOrDie(strings.Split("iptables -t nat -A PREROUTING -i "+tun+" -j DNAT --to "+ip", " ")...)
		runOrDie(strings.Split("ip route add default dev "+tun+" table 10", " ")...)
		runOrDie(strings.Split("ip rule add fwmark 1 table 10", " ")...)
		runOrDie(strings.Split("ip route flush cache", " ")...)

		// restore website redirections
		runOrDie(strings.Split("/sbin/iptables -A PREROUTING -t nat -i eth0 -p tcp --dport 80 -j REDIRECT --to-port "+*webDestPort, " ")...)
		runOrDie(strings.Split("/sbin/iptables -A PREROUTING -t nat -i eth0 -p tcp --dport 443 -j REDIRECT --to-port "+*webDestPortTLS, " ")...)

		// TODO(mpl): option to skip restoring torrent binding


	}
}

func getBoundIP() (string, error) {
	args := []string{"localhost:5000", "get_bind"}
	cmd := exec.Command(*bin, args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasSuffix(string(out), "connection refused") {
			// it's ok, rtorrent not running
			return "", noRtorrentErr
		}
		return "", err
	}
	ip := parseResponse(string(out))
	if ip == "" {
		// TODO(mpl): previous code suggests that could happen ? when rtorrent too busy mebbe ?
		return "", fmt.Errorf("bound IP not found in output: %q", string(out))
	}
	return ip, nil
}

// TODO(mpl): do it with regexp

const (
	posHint = "<param><value><string>"
	endHint = "</string></value></param>"
)

func parseResponse(xml string) string {
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

func resetBoundIP() error {
	ip, err := getBoundIP()
	if err != nil {
		if err == noRtorrentErr {
			println("rtorrent not running, that's ok")
			return nil
		}
		return err
	}
	if ip == boundIP 


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
