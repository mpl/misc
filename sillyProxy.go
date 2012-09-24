package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

const (
	idstring       = "http://golang.org/pkg/http/#ListenAndServe"
)

var (
	host       = flag.String("host", "localhost:8080", "listening port and hostname")
	camliHost  = flag.String("camlihost", "localhost:3179", "listening port and hostname for camlistored")
	help       = flag.Bool("h", false, "show this help")
	camliProxy http.Handler
)

func usage() {
	fmt.Fprintf(os.Stderr, "\t camliwrap \n")
	flag.PrintDefaults()
	os.Exit(2)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
		}()
		title := r.URL.Path
		w.Header().Set("Server", idstring)
		fn(w, r, title)
	}
}

func dasHandler(w http.ResponseWriter, r *http.Request, urlnotpackage string) {
	camliProxy.ServeHTTP(w, r)
	return
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

	camliProxyUrl, _ := url.Parse("http://" + *camliHost)
	camliProxy = httputil.NewSingleHostReverseProxy(camliProxyUrl)

	http.HandleFunc("/", makeHandler(dasHandler))
	http.ListenAndServe(*host, nil)
}
