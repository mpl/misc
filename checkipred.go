package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
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
	interval  = flag.Int("interval", 60, "Interval between runs, in seconds. use 0 to run only once.")
	resetRtorrent = flag.Bool("rtorrent", true, "Whether to reset rtorrent's bound ip (with rtorrentrpc)")
	webDestPort = flag.String("webport", "8080", "port that will get all packets destined to port 80")
	webDestPortTLS = flag.String("webportTLS", "4443", "port that will get all packets destined to port 443")
	verbose = flag.Bool("v", true, "be verbose")
)

const (
	tun = "tun100"
	noTunMsg = tun +": error fetching interface information: Device not found"
)

var (
	ipredIP        string
	boundIP string
	noTunErr = errors.New(noTunMsg)
	noRtorrentErr = errors.New("rtorrent not running")
	rtorrentrpc = "rtorrentrpc"
)

func printf(format string, args ...interface{}) {
//	log.Printf(format, args...)
	// TODO(mpl): why the fuck can't I enable *verbose from the CLI ?
	if *verbose {
		log.Printf(format, args...)
	}
}

func getTunIP() (string, error) {
	printf("getting tun IP")
	// TODO(mpl): can probably be done with the stdlib.
	cmd := exec.Command("/sbin/ifconfig", tun)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), noTunMsg) {
			return "", noTunErr
		}
		return "", fmt.Errorf("%v: %v", err, string(out))
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
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
	return "", errors.New("inet addr not found in ifconfig output")
}

func runOrDie(args ...string) {
	printf("running command: %v", args)
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil || string(out) != "" {
		killVPN()
		log.Fatalf("%v: %v", err, string(out))
	}	
}

func getBoundIP() (string, error) {
	if _, err := exec.LookPath(rtorrentrpc); err != nil {
		return "", err
	}
	cmd := exec.Command(rtorrentrpc, "localhost:5000", "get_bind")
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

func setBoundIP(giveup time.Duration) error {
	// TODO(mpl): use my lib xml-rpc instead of rtorrentrpc
	// first make sure we have rtorrentrpc so we can return early if not
	if _, err := exec.LookPath(rtorrentrpc); err != nil {
		return err
	}
	args := []string{"localhost:5000", "set_bind", ipredIP}
	retryPause := 1 * time.Second
	stop := time.Now().Add(giveup)
	// TODO(mpl): use a time.Timer or whatever's efficient these days
	for {
		if time.Now().After(stop) {
			return fmt.Errorf("giving up resetting bound IP after %v", giveup)
		}
		cmd := exec.Command(rtorrentrpc, args...)
		out, err := cmd.CombinedOutput()
		if err == nil && string(out) != "" {
			boundIP = ipredIP
			return nil
		}
		time.Sleep(retryPause)
		retryPause = retryPause * 2
	}
}

func resetBoundIP() error {
	printf("resetting bound IP to %v", ipredIP)	
	ip, err := getBoundIP()
	if err != nil {
		if err == noRtorrentErr {
			println("rtorrent not running, that's ok")
			return nil
		}
		return err
	}
	if ip == ipredIP {
		return nil
	}
	return setBoundIP(10*time.Minute)
}


// because then I can have a defer to sleep
func mainLoop() error {
		ip, err := getTunIP()
		if err != nil {
			if err != noTunErr {
				// TODO(mpl): warn me
				return err
			}
			runOrDie(strings.Fields("/usr/sbin/service openvpn start ipredator")...)
			return nil
		}
		if ip == ipredIP {
			printf("current tun IP == ipredIP (%v)", ipredIP)
			if ip == boundIP {
				printf("current tun IP == boundIP (%v). all good.", ipredIP)	
				return nil
			}
			if err := resetBoundIP(); err != nil {
				return err
			}			
		}
		ipredIP = ip

		// mark packets that should go through the tunnel
		runOrDie(strings.Fields("iptables -t nat -F")...)
		runOrDie(strings.Fields("iptables -t mangle -F")...)
		runOrDie(strings.Fields("iptables -t mangle -A OUTPUT --source "+ip+" -j MARK --set-mark 1")...)
		runOrDie(strings.Fields("iptables -t nat -A POSTROUTING -o "+tun+" -j SNAT --to "+ip)...)
		runOrDie(strings.Fields("iptables -t nat -A PREROUTING -i "+tun+" -j DNAT --to "+ip)...)
		runOrDie(strings.Fields("ip route add default dev "+tun+" table 10")...)
		runOrDie(strings.Fields("ip rule add fwmark 1 table 10")...)
		runOrDie(strings.Fields("ip route flush cache")...)

		// restore website redirections
		runOrDie(strings.Split("/sbin/iptables -A PREROUTING -t nat -i eth0 -p tcp --dport 80 -j REDIRECT --to-port "+*webDestPort, " ")...)
		runOrDie(strings.Split("/sbin/iptables -A PREROUTING -t nat -i eth0 -p tcp --dport 443 -j REDIRECT --to-port "+*webDestPortTLS, " ")...)

		if !*resetRtorrent {
			return nil
		}
		if err := resetBoundIP(); err != nil {
			return err
		}
	return nil
}

func killVPN() {
	cmd := exec.Command("/usr/sbin/service", "openvpn", "stop", "ipredator")
	var buff bytes.Buffer
	cmd.Stderr = &buff
	err := cmd.Run()
	stderr := buff.String()
	if err != nil || stderr != "" {
		// TODO(mpl): warn me
		log.Printf("could not stop vpn: %v: %v", err, stderr)
	}	
}

func main() {
	for {
		if err := mainLoop(); err != nil {
			killVPN()
			log.Fatal(err)
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

	/*
func main() {
	flag.Parse()
	checkFlags()
	ipredIP = flag.Args()[0]
	checkBinding()

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
}
	*/
