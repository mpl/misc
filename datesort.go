/*
Copyright 2013 Mathieu Lonjaret.

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
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

const outDir = "timeSorted"

func main() {
	flag.Parse()
	args := flag.Args()
	sortedNames := sortByTime(args)
	renameSorted(sortedNames)
}

func renameSorted(sorted []string) error {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}
	// TODO(mpl): how to preset the formatting?
	l := len(sorted)
	var filename string
	for k, v := range sorted {
		switch {
		case l < 10:
			filename = fmt.Sprintf("%d.jpg", k+1)
		case l < 100:
			filename = fmt.Sprintf("%02d.jpg", k+1)
		case l < 1000:
			filename = fmt.Sprintf("%03d.jpg", k+1)
		default:
			panic("more than 10000 pics. you crazy.")
		}
		newName := filepath.Join(outDir, filename)
		f, err := os.Open(v)
		if err != nil {
			return err
		}
		// not using defer to close because we can run out of file handlers
		g, err := os.OpenFile(newName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("Could not create %v: %v", newName, err)
			f.Close()
		}
		_, err = io.Copy(g, f)
		if err != nil {
			f.Close()
			g.Close()
			return err
		}
		f.Close()
		g.Close()
	}
	return nil
}

// It would take less memory overall to make a type that holds both the name and the time, and to do a sort on that. but I'm lazy right now.
func sortByTime(names []string) []string {
	timeToName := make(map[string]string)
	var times []string
	for _, v := range names {
		f, err := os.Open(v)
		if err != nil {
			log.Print(err)
			continue
		}
		defer f.Close()
		dt, err := DecodeDate(f)
		if err != nil {
			log.Print(err)
			continue
		}
		timeToName[dt] = v
		times = append(times, dt)
	}
	sort.Strings(times)
	var sortedNames []string
	for _, v := range times {
		sortedNames = append(sortedNames, timeToName[v])
	}
	return sortedNames
}

func DecodeDate(r io.Reader) (string, error) {
	var t string
	lr := io.LimitReader(r, 2<<20)

	ex, err := exif.Decode(lr)
	if err != nil {
		return t, fmt.Errorf("No valid EXIF.")
	}
	date, err := ex.Get(exif.DateTimeOriginal)
	if err != nil {
		date, err = ex.Get(exif.DateTime)
		if err != nil {
			return t, err
		}
	}
	if date.Type != 2 {
		return t, errors.New("DateTime[Original] not in string format")
	}
//	exifTimeLayout := "2006:01:02 15:04:05"
//	dateStr := strings.TrimRight(date.StringVal(), "\x00")
//	return time.Parse(exifTimeLayout, dateStr)
	return strings.TrimRight(date.StringVal(), "\x00"), nil
}
