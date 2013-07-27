package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	artist, album, title, track string
	quality                     int
	oggOptions                  = map[string]string{
		"artist":      "-a",
		"album":       "-l",
		"title":       "-t",
		"tracknumber": "-n",
	}
)

func init() {
	flag.StringVar(&artist, "artist", "", "artist metadata")
	flag.StringVar(&album, "album", "", "album metadata")
	flag.StringVar(&title, "title", "", "title metadata")
	flag.StringVar(&track, "track", "", "track number metadata")
	flag.IntVar(&quality, "quality", 3, "quality level for oggenc")
}

func doit(fullpath string) error {
	tags, err := meta(fullpath)
	if err != nil {
		return err
	}
	err = convert(fullpath, tags)
	return err
}

func convert(fullpath string, tags map[string]string) error {
	cmd1 := exec.Command("flac", "-d", "-c", fullpath)
	stdout, err := cmd1.StdoutPipe()
	if err != nil {
		return err
	}
	dir, filename := filepath.Split(fullpath)
	if !strings.HasSuffix(filename, ".flac") {
		return fmt.Errorf("filename %v does not end in \".ogg\", giving up.", filename)
	}
	outfile := filepath.Join(dir,
		strings.Replace(filename, ".flac", ".ogg", 1))
	args := []string{"-", "-q", fmt.Sprintf("%d", quality), "-o", outfile}
	for k, v := range tags {
		if v != "" {
			args = append(args, oggOptions[k], v)
		}
	}
	cmd2 := exec.Command("oggenc", args...)
	stdin, err := cmd2.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd1.Start(); err != nil {
		return fmt.Errorf("Could not start flac: %v", err)
	}
	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("Could not start oggenc: %v", err)
	}
	_, err = io.Copy(stdin, stdout)
	if err != nil {
		return fmt.Errorf("Could not pipe: %v", err)
	}
	return nil
}

func meta(fullpath string) (map[string]string, error) {
	cmdname := "metaflac"
	tags := make(map[string]string)
	for k, v := range map[string]string{
		"artist":      artist,
		"album":       album,
		"title":       title,
		"tracknumber": track} {
		if v == "" {
			upper := strings.ToUpper(k)
			args := []string{"--show-tag=" + upper, fullpath}
			cmd := exec.Command(cmdname, args...)
			output, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("Could not run metaflac: %v", err)
			}
			tag := strings.Replace(string(output), upper+"=", "", 1)
			tag = strings.TrimSuffix(tag, "\n")
			tags[k] = tag
			continue
		}
		tags[k] = v
	}
	return tags, nil
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		os.Exit(1)
	}
	err := doit(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
