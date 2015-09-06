package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	interval       = flag.Int("interval", 60, "Interval between runs, in seconds. use 0 to run only once.")
	resetRtorrent  = flag.Bool("rtorrent", true, "Whether to reset rtorrent's bound ip (with rtorrentrpc)")
	webDestPort    = flag.String("webport", "8080", "port that will get all packets destined to port 80")
	webDestPortTLS = flag.String("webportTLS", "4443", "port that will get all packets destined to port 443")
	host           = flag.String("host", "", "host to check to see if routing is all good")
	logfile        = flag.String("logfile", "", "write there in addition to stdout/stderr")
	verbose        = flag.Bool("v", true, "be verbose")
)

const (
	tun           = "tun100"
	noTunMsg      = tun + ": error fetching interface information: Device not found"
	isRoutingHint = "default dev " + tun + "  scope link"
)

var (
	noTunErr      = errors.New(noTunMsg)
	noRtorrentErr = errors.New("rtorrent not running")
	rtorrentrpc   = "rtorrentrpc"
	giveup        = 10 * time.Minute
)

func printf(format string, args ...interface{}) {
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
		l := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(l, "inet addr:") {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) != 4 {
			return "", fmt.Errorf("wrong number of parts in inet addr line")
		}
		return strings.TrimSpace(strings.TrimPrefix(parts[1], "addr:")), nil
	}
	return "", errors.New("inet addr not found in ifconfig output")
}

func stringCmd(cmd string) *exec.Cmd {
	fields := strings.Fields(cmd)
	return exec.Command(fields[0], fields[1:]...)
}

func run(args ...string) error {
	printf("running command: %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil || stderr.Len() != 0 {
		return fmt.Errorf("%v: %v", err, stderr.String())
	}
	return nil
}

func runNoOutput(args ...string) error {
	printf("running command: %v", args)
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil || string(out) != "" {
		return fmt.Errorf("%v: %v", err, string(out))
	}
	return nil
}

func getBoundIP(giveup time.Duration) (string, error) {
	if _, err := exec.LookPath(rtorrentrpc); err != nil {
		return "", err
	}
	args := []string{"localhost:5000", "get_bind"}
	retryPause := 1 * time.Second
	stop := time.Now().Add(giveup)
	// TODO(mpl): use a time.Timer or whatever's efficient these days
	var lastErr error
	for {
		if time.Now().After(stop) {
			return "", fmt.Errorf("giving up getting bound IP after %v: %v", giveup, lastErr)
		}
		cmd := exec.Command(rtorrentrpc, args...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			// TODO(mpl): Getting the "EOF" message from rtorrentrpc (too busy?) is super lame, I should fix rtorrentrpc
			if !strings.HasSuffix(strings.TrimSpace(string(out)), "connection refused") {
				time.Sleep(retryPause)
				retryPause = retryPause * 2
				lastErr = fmt.Errorf("rtorrentrpc error: %v, %v", err, string(out))
				continue
			}
			// it's ok, rtorrent not running
			printf("rtorrent not running, that's ok")
			return "", noRtorrentErr
		}
		ip := parseResponse(string(out))
		if ip == "" {
			time.Sleep(retryPause)
			retryPause = retryPause * 2
			lastErr = fmt.Errorf("bound IP not found in output: %q", string(out))
			continue
		}
		return ip, nil
	}
}

// TODO(mpl): do it with regexp

const (
	posHint = "<param><value><string>"
	endHint = "</string></value></param>"
)

func parseResponse(xml string) string {
	idx := strings.Index(xml, "<param><value><string>")
	if idx <= 0 {
		printf("error while parsing bound ip response: no beg pos")
		return ""
	}
	begin := idx + len(posHint)
	xml = xml[begin:]
	idx = strings.Index(xml, endHint)
	if idx <= 0 {
		printf("error while parsing bound ip response: no end pos")
		return ""
	}
	return xml[:idx]
}

func setBoundIP(newIP string, giveup time.Duration) error {
	// TODO(mpl): use my lib xml-rpc instead of rtorrentrpc
	// first make sure we have rtorrentrpc so we can return early if not
	if _, err := exec.LookPath(rtorrentrpc); err != nil {
		return err
	}
	args := []string{"localhost:5000", "set_bind", newIP}
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
			return nil
		}
		time.Sleep(retryPause)
		retryPause = retryPause * 2
	}
}

func resetBoundIP(newIP string) error {
	ip, err := getBoundIP(giveup)
	if err != nil {
		if err == noRtorrentErr {
			printf("rtorrent not running, that's ok")
			return nil
		}
		return err
	}
	if ip == newIP {
		return nil
	}
	printf("resetting bound IP to %v", newIP)
	return setBoundIP(newIP, giveup)
}

func isRouting(ip string) bool {
	out, err := stringCmd("iptables -t nat -n -L PREROUTING 1").Output()
	strOut := strings.TrimSpace(string(out))
	if err != nil || strOut == "" {
		return false
	}
	fields := strings.Fields(strOut)
	if len(fields) != 6 {
		printf("wrong number of fields when checking routing")
		return false
	}
	if strings.TrimPrefix(fields[5], "to:") != ip {
		return false
	}
	return true
}

func setRouting(ip string) error {
	printf("updating routing with %v", ip)
	var stickyErr error
	checkErr := func(args string) {
		if stickyErr != nil {
			printf("fallthrough because stickyErr")
			return
		}
		stickyErr = runNoOutput(strings.Fields(args)...)
	}
	// mark packets that should go through the tunnel
	checkErr("iptables -t nat -F")
	checkErr("iptables -t mangle -F")
	checkErr("iptables -t mangle -A OUTPUT --source " + ip + " -j MARK --set-mark 1")
	checkErr("iptables -t nat -A POSTROUTING -o " + tun + " -j SNAT --to " + ip)
	checkErr("iptables -t nat -A PREROUTING -i " + tun + " -j DNAT --to " + ip)
	checkErr("ip route add default dev " + tun + " table 10")
	checkErr("ip rule add fwmark 1 table 10")
	checkErr("ip route flush cache")

	// restore website redirections
	checkErr("/sbin/iptables -A PREROUTING -t nat -i eth0 -p tcp --dport 80 -j REDIRECT --to-port " + *webDestPort)
	checkErr("/sbin/iptables -A PREROUTING -t nat -i eth0 -p tcp --dport 443 -j REDIRECT --to-port " + *webDestPortTLS)
	return stickyErr
}

func mainLoop() error {
	ip, err := getTunIP()
	if err != nil {
		if err != noTunErr {
			return err
		}
		if err := run(strings.Fields("/usr/sbin/service openvpn start ipredator")...); err != nil {
			return err
		}
		time.Sleep(10 * time.Second)
		ip, err = getTunIP()
		if err != nil {
			return err
		}
	}

	if !isRouting(ip) {
		if err := setRouting(ip); err != nil {
			return err
		}
	}

	if *host != "" {
		resp, err := http.Get(*host)
		if err != nil || resp.StatusCode != 200 {
			return fmt.Errorf("could not reach %v: %v", *host, err)
		}
		defer resp.Body.Close()
	}

	if !*resetRtorrent {
		return nil
	}
	return resetBoundIP(ip)
}

func killVPN() {
	printf("killing vpn")
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
	flag.Parse()
	if *logfile != "" {
		f, err := os.Create(*logfile)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
		}()
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	}

	// We could remove that loop and use cron instead BUT, I don't want risking cron starting an instance while a previous one is still running, hence why we control it from here.
	for {
		if err := mainLoop(); err != nil {
			printf("%v", err)
			killVPN()
		}
		if *interval <= 0 {
			return
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}
