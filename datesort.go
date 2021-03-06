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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

const outDir = "timeSorted"

var (
	check = flag.Bool("check", false, "Check with md5sum that all initial files end up in the out dir")
	dry   = flag.Bool("dry", false, "Do not actually write the renamed files.")
)

func main() {
	flag.Parse()
	args := flag.Args()
	sortedNames := sortByTime(args)
	if !*dry {
		renameSorted(sortedNames)
	}
	if *check {
		finalCheck(args)
	}
}

// make it do things concurrently?
func finalCheck(inputFiles []string) {
	f, err := os.Open(outDir)
	if err != nil {
		log.Fatalf("Could not open %v: %v", outDir, err)
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		log.Fatalf("Could not read dir names for %v: %v", outDir, err)
	}
	if len(names) != len(inputFiles) {
		log.Printf("Number of input args: %d - number of files in %v: %d", len(inputFiles), outDir, len(names))
	}

	renamedHashes := make(map[string]string)
	for _, v := range names {
		cmd := exec.Command("md5sum", filepath.Join(outDir, v))
		out, err := cmd.Output()
		if err != nil {
			log.Fatalf("Could not exec %v: %v", out, err)
		}
		sum := strings.SplitN(string(out), " ", 2)[0]
		//		println(v + ": " + sum)
		renamedHashes[v] = sum
	}

	inputHashes := make(map[string]string)
	for _, v := range inputFiles {
		cmd := exec.Command("md5sum", v)
		out, err := cmd.Output()
		if err != nil {
			log.Fatalf("Could not exec md5sum %v: %v", v, err)
		}
		sum := strings.SplitN(string(out), " ", 2)[0]
		//		println(v + ": " + sum)
		inputHashes[v] = sum
	}

	for inputFile, inputHash := range inputHashes {
		found := ""
		for renamed, renamedHash := range renamedHashes {
			if inputHash == renamedHash {
				found = renamed
				break
			}
		}
		if found == "" {
			log.Printf("%v was not found in the output files", inputFile)
		} else {
			delete(renamedHashes, found)
		}
	}
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
		case l < 10000:
			filename = fmt.Sprintf("%04d.jpg", k+1)
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
	for _, name := range names {
		f, err := os.Open(name)
		if err != nil {
			log.Print(err)
			continue
		}
		dt, err := DecodeDate(f)
		f.Close()
		if err != nil {
			log.Printf("%v: %v", name, err)
			continue
		}
		if sametime, ok := timeToName[dt]; ok {
			// this can happen because DateTime is apparently only precise up to the second
			log.Printf("%v and %v have the same time %v. now appending suffix to time key for uniqueness.", sametime, name, dt)
			i := 1
			for {
				newtime := fmt.Sprintf("%s-%d", dt, i)
				if _, ok := timeToName[newtime]; !ok {
					// does not exist, all good
					dt = newtime
					break
				}
				// already exists, increase suffix.
				i++
			}
		}
		timeToName[dt] = name
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
