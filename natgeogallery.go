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

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	flagVerbose = flag.Bool("v", false, "be verbose")
	flagDry     = flag.Bool("dry", false, "do not actually fetch the images")
	flagHelp    = flag.Bool("h", false, "shows this help")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: natgeogallery pageToParse baseURL\n")
	fmt.Fprintf(os.Stderr, "Fetch all images found in the source of pageToParse that start with pattern baseURL\n.")
	fmt.Fprintf(os.Stderr, "Example: natgeogallery http://news.nationalgeographic.com/2016/01/160123-snowzilla-blizzard-snowstorm-us-northeast-pictures/ http://news.nationalgeographic.com/content/dam/news/2016/01/23/snow_gallery/\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *flagHelp {
		usage()
	}
	args := flag.Args()
	if len(args) != 2 {
		usage()
	}

	res, err := http.Get(args[0])
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	srcHTML, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	imgURL := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(srcHTML))

	galleryPrefix := args[1]
	galleryPattern := regexp.MustCompile(`.*data-src="(` + galleryPrefix + `.*?\.jpg).*"`)
	for scanner.Scan() {
		l := scanner.Text()
		if !strings.Contains(l, "data-src") {
			continue
		}
		m := galleryPattern.FindStringSubmatch(l)
		if m == nil || len(m) != 2 {
			continue
		}
		if *flagVerbose {
			log.Printf("Found a match: %v", m[1])
		}
		imgURL[m[1]] = true
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	if *flagDry {
		return
	}
	i := 1
	for k, _ := range imgURL {
		if *flagVerbose {
			log.Printf("Fetching %v", k)
		}
		res, err := http.Get(k)
		if err != nil {
			log.Printf("failed to fetch img %v: %v", k, err)
			continue
		}
		defer res.Body.Close()
		f, err := os.Create(fmt.Sprintf("%d.jpg", i))
		if err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(f, res.Body); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		i++
	}

}
