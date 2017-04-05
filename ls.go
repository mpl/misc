/*
Copyright 2017 Mathieu Lonjaret

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

// static "ls" binary so I can debug FROM SCRATCH docker images.
package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	flag.Parse()
	args := flag.Args()
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	f, err := os.Open(target)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range names {
		println(v)
	}
}
