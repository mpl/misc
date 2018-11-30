package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

//	"golang.org/x/net/http2"
)

const (
	idstring   = "http://golang.org/pkg/http/#ListenAndServe"
)

// subdomains we reverse proxy to.
var (
	hosts []string

	pkProxy   http.Handler
	picsProxy    http.Handler
	websiteProxy http.Handler
)

var (
	help    = flag.Bool("h", false, "show this help")
	flagDebug   = flag.Bool("v", false, "log some stuff")
	flagHost = flag.String("host", ":80", "host:port on which to listen")

	logger *log.Logger
)

func usage() {
	fmt.Fprintf(os.Stderr, "\t reverseproxy \n")
	flag.PrintDefaults()
	os.Exit(2)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				log.Printf("WARNING: %v", e.Error())
				http.Error(w, "Internal error, please bug me about it.", http.StatusInternalServerError)
				return
			}
		}()
		w.Header().Set("Server", idstring)
		fn(w, r)
	}
}

func pkHandler(w http.ResponseWriter, r *http.Request) {
	logger.Printf("for perkeep proxy: %v", r.URL.Path)
	pkProxy.ServeHTTP(w, r)
}

func picsHandler(w http.ResponseWriter, r *http.Request) {
	logger.Printf("for pics proxy: %v", r.URL.Path)
	picsProxy.ServeHTTP(w, r)
}

func websiteHandler(w http.ResponseWriter, r *http.Request) {
	logger.Printf("for website proxy: %v", r.URL.Path)
	websiteProxy.ServeHTTP(w, r)
}

func initProxies() {
	frontEndHost := "home.granivo.re"

	pkHost := "pk." + frontEndHost
	frontEndBaseURL := "http://" + frontEndHost
	hosts = []string{frontEndHost, pkHost}

	picsBaseURL := frontEndBaseURL + ":3178"
	picsProxyUrl, _ := url.Parse(picsBaseURL)
	picsProxy = newSingleHostReverseProxy(picsProxyUrl)
	http.HandleFunc(pkHost+"/pics/", makeHandler(picsHandler))
	http.HandleFunc(frontEndHost+"/pics/", makeHandler(picsHandler))

	pkBaseURL := "http://" + pkHost + ":3178"
	pkProxyUrl, _ := url.Parse(pkBaseURL)
	pkProxy = newSingleHostReverseProxy(pkProxyUrl)
	http.HandleFunc(pkHost+"/", makeHandler(pkHandler))

	websiteBaseURL := frontEndBaseURL + ":4430"
	websiteProxyUrl, _ := url.Parse(websiteBaseURL)
	websiteProxy = newSingleHostReverseProxy(websiteProxyUrl)
	http.HandleFunc("/", makeHandler(websiteHandler))

}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *help {
		usage()
	}

	nargs := flag.NArg()
	if nargs > 0 {
		usage()
	}

	if *flagDebug {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}

	initProxies()

	ln, err := net.Listen("tcp", *flagHost)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := &http.Server{
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout: 120 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil {
			log.Fatal(err)
		}
	}()

	select {}
}

// reverseProxy is an HTTP Handler that takes an incoming request and
// sends it to another server, proxying the response back to the
// client.
type reverseProxy struct {
	// Director must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Director must not access the provided Request
	// after returning.
	director func(*http.Request)

	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	transport http.RoundTripper

	// BufferPool optionally specifies a buffer pool to
	// get byte slices for use by io.CopyBuffer when
	// copying HTTP response bodies.
	BufferPool BufferPool
}

// A BufferPool is an interface for getting and returning temporary
// byte slices for use by io.CopyBuffer.
type BufferPool interface {
	Get() []byte
	Put([]byte)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// newSingleHostReverseProxy returns a new ReverseProxy that routes
// URLs to the scheme, host, and base path provided in target. If the
// target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
func newSingleHostReverseProxy(target *url.URL) *reverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	return &reverseProxy{director: director}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func cloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

func (p *reverseProxy) handleError(rw http.ResponseWriter, req *http.Request, err error) {
	p.logf("http: proxy error: %v", err)
	rw.WriteHeader(http.StatusBadGateway)
}

func (p *reverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	outreq := req.WithContext(ctx) // includes shallow copies of maps, but okay
	if req.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	outreq.Header = cloneHeader(req.Header)
	p.director(outreq)
	outreq.Close = false
	removeConnectionHeaders(outreq.Header)

	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range hopHeaders {
		hv := outreq.Header.Get(h)
		if hv == "" {
			continue
		}
		outreq.Header.Del(h)
	}

	res, err := http.DefaultTransport.RoundTrip(outreq)
	if err != nil {
		p.handleError(rw, outreq, err)
		return
	}
	removeConnectionHeaders(res.Header)
	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	copyHeader(rw.Header(), res.Header)
	rw.WriteHeader(res.StatusCode)
	err = p.copyResponse(rw, res.Body)
	defer res.Body.Close()
	if err != nil {
		panic(http.ErrAbortHandler)
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeConnectionHeaders(h http.Header) {
	if c := h.Get("Connection"); c != "" {
		for _, f := range strings.Split(c, ",") {
			if f = strings.TrimSpace(f); f != "" {
				h.Del(f)
			}
		}
	}
}

func (p *reverseProxy) copyResponse(dst io.Writer, src io.Reader) error {
	var buf []byte
	if p.BufferPool != nil {
		buf = p.BufferPool.Get()
		defer p.BufferPool.Put(buf)
	}
	_, err := p.copyBuffer(dst, src, buf)
	return err
}

// copyBuffer returns any write errors or non-EOF read errors, and the amount
// of bytes written.
func (p *reverseProxy) copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			p.logf("httputil: ReverseProxy read error during body copy: %v", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

func (p *reverseProxy) logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
