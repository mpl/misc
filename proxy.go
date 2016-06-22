/*
Copyright 2016 Mathieu Lonjaret

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Quick and dirty proxy to Camlistore in docker. To simulate e.g. nginx between
// Camlistore and the rest of the world.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/mpl/basicauth"
)

const (
	idstring   = "http://golang.org/pkg/http/#ListenAndServe"
)

var (
	host     = flag.String("host", "0.0.0.0:3179", "listening port and hostname")
	help     = flag.Bool("h", false, "show this help")
	flagUserpass = flag.String("userpass", "one:two", "optional username:password protection")
	flagTLS   = flag.Bool("tls", false, `For https. If "key.pem" or "cert.pem" are not found in $HOME/keys/, in-memory self-signed are generated and used instead.`)
)

var (
	rootdir, _        = os.Getwd()
	up *basicauth.UserPass
)

func usage() {
	fmt.Fprintf(os.Stderr, "\t proxy \n")
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
		if isAllowed(r) {
			println("PROXY: IS ALLOWED")
			fn(w, r, title)
		} else {
			println("PROXY: NOT ALLOWED")
			basicauth.SendUnauthorized(w, r, "proxy NOPE")
		}
	}
}

func isAllowed(r *http.Request) bool {
	return up.IsAllowed(r)
}

func initUserPass() {
	if *flagUserpass == "" {
		return
	}
	var err error
	up, err = basicauth.New(*flagUserpass)
	if err != nil {
		log.Fatal(err)
	}
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

	initUserPass()

	proxyURL, err := url.Parse("http://172.17.0.2:3179/")
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)

	http.Handle("/", makeHandler(func(w http.ResponseWriter, r *http.Request, whatever string) {
		proxy.ServeHTTP(w, r)
	}))
	if err = http.ListenAndServe(":3179", nil); err != nil {
		log.Fatalf("Error in http server: %v\n", err)
	}
}
