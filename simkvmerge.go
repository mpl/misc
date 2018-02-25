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
	"regexp"
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
		skip, err := merge(name)
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		if skip {
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
			log.Printf("%v done. %d/%d directories done.", name, done, len(names))
		}
	}
}

func merge(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()
	fi, err := dir.Stat()
	if err != nil {
		return false, err
	}
	if !fi.IsDir() {
		return true, nil
	}
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return false, err
	}

	flim := ""
	sub := ""
	outFile := "out.mkv"
	for _, name := range names {
		if name == outFile {
			log.Printf("%v already contains %v, skipping merging for this dir", dirPath, outFile)
			return false, nil
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
		return false, fmt.Errorf("no flim found in dir %v", dirPath)
	}
	fullFlimPath := filepath.Join(dirPath, flim)
	hasFrenchSub, err := hasFrSub(fullFlimPath)
	if err != nil {
		return false, fmt.Errorf("error while scanning for french subs: %v", err)
	}
	if hasFrenchSub {
		if *flagVerbose {
			log.Printf("Skipping merging for %v because it already has french subs", fullFlimPath)
		}
		return true, nil
	}
	if sub == "" {
		return false, fmt.Errorf("no sub found in dir %v", dirPath)
	}
	args := []string{
		"-o", filepath.Join(dirPath, outFile),
		fullFlimPath,
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
		return false, fmt.Errorf("mkvmerge error: %v, %v", err, buf.String())
	}
	return false, nil
}

var frSubRxp = regexp.MustCompile(`^    Stream #\d:\d\(fre\): Subtitle:.*`)

func hasFrSub(flimPath string) (bool, error) {
	cmd := exec.Command("ffmpeg", "-i", flimPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// TODO(mpl): this is to ignore the error from ffmpeg that we get because we do
		// not give an output file. It is disgusting. We should instead use another
		// command, or maybe some obscure ffmpeg option.
		if !bytes.HasSuffix(out, []byte("At least one output file must be specified\n")) {
			return false, fmt.Errorf("ffmpeg error for %v: %v, %v", flimPath, err, string(out))
		}
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		l := sc.Text()
		if frSubRxp.MatchString(l) {
			return true, nil
		}
	}
	return false, sc.Err()
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
