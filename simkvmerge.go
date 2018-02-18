package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	flagVerbose        = flag.Bool("v", false, "be verbose")
	flagRemoveOriginal = flag.Bool("remove_original", false, "on success, overwrite the old original file with the newly produced one (out.mkv)")
)

func main() {
	flag.Parse()

	dir, err := os.Open(".")
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	done := 0
	for _, name := range names {
		if err := merge(name); err != nil {
			log.Printf("%v", err)
			continue
		}
		if *flagRemoveOriginal {
			if err := overwriteOriginal(name); err != nil {
				log.Printf("%v", err)
				continue
			}
		}
		if *flagVerbose {
			done++
			log.Printf("%v done. %d/%d directories done.", done, len(names))
		}
	}
}

func merge(dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	fi, err := dir.Stat()
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return nil
	}
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	flim := ""
	sub := ""
	outFile := "out.mkv"
	for _, name := range names {
		if name == outFile {
			log.Printf("%v already contains %v, skipping this dir", dirPath, outFile)
			return nil
		}
		// TODO(mpl): take into account more extensions
		if flim == "" && strings.HasSuffix(name, ".mkv") {
			flim = name
		}
		if sub == "" && strings.HasSuffix(name, ".srt") {
			sub = name
		}
	}
	if flim == "" {
		return fmt.Errorf("no flim found in dir %v", dirPath)
	}
	if sub == "" {
		return fmt.Errorf("no sub found in dir %v", dirPath)
	}
	args := []string{
		"-o", filepath.Join(dirPath, outFile),
		filepath.Join(dirPath, flim),
		"--language", "0:fre",
		"--track-name", "0:Forced",
		"--forced-track", "0:yes",
		"--default-track", "0:yes",
		filepath.Join(dirPath, sub),
	}
	cmd := exec.Command("mkvmerge", args...)
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mkvmerge error: %v, %v", err, buf.String())
	}
	return nil
}

func overwriteOriginal(dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	fi, err := dir.Stat()
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return nil
	}
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	flim := ""
	outFile := "out.mkv"
	outFound := false
	for _, name := range names {
		if name == outFile {
			outFound = true
		}
		// TODO(mpl): take into account more extensions
		if flim == "" && name != outFile && strings.HasSuffix(name, ".mkv") {
			flim = name
		}
	}
	if flim == "" {
		return fmt.Errorf("no flim found in dir %v", dirPath)
	}
	if !outFound {
		if *flagVerbose {
			log.Printf("No %v found in %v, nothing to overwrite", outFile, dirPath)
		}
		return nil
	}

	flimFullpath := filepath.Join(dirPath, flim)
	cmd := exec.Command("mkvmerge", "-i", flimFullpath)
	var buf bytes.Buffer
	cmd.Stderr = &buf
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("mkvmerge error for %v: %v, %v", flimFullpath, err, buf.String())
	}
	oriCount, err := countSublines(out)
	if err != nil {
		return err
	}

	outFullpath := filepath.Join(dirPath, outFile)
	cmd = exec.Command("mkvmerge", "-i", outFullpath)
	cmd.Stderr = &buf
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("mkvmerge error for %v: %v, %v", outFullpath, err, buf.String())
	}
	outCount, err := countSublines(out)
	if err != nil {
		return err
	}

	if outCount <= oriCount {
		if *flagVerbose {
			log.Printf("%v does not have more subtitles than %v; refusing to overwriting %v", outFullpath, flimFullpath, flimFullpath)
		}
		return nil
	}
	if err := os.Rename(outFullpath, flimFullpath); err != nil {
		return err
	}
	if *flagVerbose {
		log.Printf("%v successfully overwritten", flimFullpath)
	}
	return nil
}

func countSublines(input []byte) (int, error) {
	count := 0
	sc := bufio.NewScanner(bytes.NewReader(input))
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "subtitles") {
			count++
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return count, nil

}
