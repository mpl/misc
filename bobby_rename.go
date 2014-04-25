package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

const (
	marker = "Kom.Nr"
	logFile = "errors.log"
)

var (
	dryrun = flag.Bool("dryrun", false, "print action, but don't actually do the renaming")
)

func main() {
	flag.Parse()
	logger, err := os.Create(logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()
	log.SetOutput(logger)

	dir, err := os.Open(".")
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	for _,name := range names {
		if !strings.HasSuffix(name, ".doc") {
			continue
		}
		kom, err := grep(name, marker)
		if err != nil {
			log.Fatalf("could not grep in %v: %v", name, err)
		}
		newName := strings.Replace(kom, "/", "_", 1)+".doc"
		if newName == name {
			continue
		}
		if *dryrun {
			log.Printf("would rename %v into %v", name, newName)
		} else {
			if err := os.Rename(name, newName); err != nil {
				log.Fatalf("error renaming %v into %v: %v", name, newName, err)
			}
		}		
	}
}

func slurp(sc *bufio.Scanner) (string, error) {
	i := 0
	slurped := ""
	for sc.Scan() {
		if i == 3 {
			break
		}
		slurped += sc.Text()
		i++
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("could not slurp after marker: %v", err)
	}
	return slurped, nil
}

var komPattern = regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+/[0-9]+)`)

func grep(filePath string, marker string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Split(bufio.ScanWords)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, marker) {
			continue
		}
		slurped, err := slurp(sc)
		if err != nil {
			return "", fmt.Errorf("could not slurp after marker: %v", err)
		}
		if !komPattern.MatchString(slurped) {
			return "", fmt.Errorf("could not find pattern after marker in %q", slurped)
		}
		return slurped, nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("could not find marker %v in %v", marker, filePath)
}
