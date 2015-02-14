package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/inotify"
	"path"
	"strings"
	"time"
)

var (
	verbose = flag.Bool("v", false, "enable verbose inotify events")
	h = flag.Bool("h", false, "shows this help")
	recursive = flag.Bool("r", false, "recursive")
	output = flag.String("o", "", "output file")
	strip = flag.String("strip", "", "path base to strip from outputs")
)

type prevEv struct {
	from bool
	name string
	when int64
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gonotify [dir1 [dir2] ...] \n")
	flag.PrintDefaults()
	os.Exit(2)
}

func addToWatcher(watcher *inotify.Watcher, dirPath string) {
	err := watcher.Watch(dirPath)
	if err != nil {
	    log.Fatal(err)
	}
	if *recursive {
		dir, err := os.Open(dirPath)
		if err != nil {
			log.Fatal(err)
		}
		names, err := dir.Readdirnames(-1)
		if err != nil {
			log.Fatal(err)
		}
		dir.Close()
		var fi *os.FileInfo
		fullPath := ""
		for _, name := range names {
			fullPath = path.Join(dirPath, name)
			fi, err = os.Lstat(fullPath)
			if err != nil {
				log.Fatal(err)
			}
			if fi.IsDirectory() {
				addToWatcher(watcher, fullPath)
			}
		}
	}
}

//TODO: detect an mkdir and add the dir to the watcher?
func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	
	if *h {
		usage()
	}
	watcher, err := inotify.NewWatcher()
	if err != nil {
	    log.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if len(args) == 0 {
		addToWatcher(watcher, cwd)
	} else {
		dirPath := ""
		temp := ""
		for _,v := range args {
			temp = path.Clean(v)
			if temp[0] != '/' {
				dirPath = path.Join(cwd, temp)
			} else {
				dirPath = temp
			}
			addToWatcher(watcher, dirPath)
		}
	}
	if len(*output) != 0 {
		f, err := os.OpenFile(*output, os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(f)
	}
	
	prev := prevEv{false, "", time.Seconds()}
	for {
	    select {
	    case ev := <-watcher.Event:
			if prev.from && time.Seconds() - prev.when > 1{
		    // that event came in too late compared to the last IN_MOVED_FROM to be the corresponding IN_MOVED_TO, hence the file was moved out of the watched locations.
		    // TODO: output the real time this happened if possible
				log.Println("removed: ", path.Clean(strings.Replace(prev.name, *strip, "", -1)))
				prev.from = false
			}	    
			switch (ev.Mask & ^inotify.IN_ISDIR) {
			case inotify.IN_CREATE:
				log.Println("new:	", path.Clean(strings.Replace(ev.Name, *strip, "", -1)))
			case inotify.IN_MOVED_TO:
				if prev.from {
					log.Println("moved:	", path.Clean(strings.Replace(prev.name, *strip, "", -1)), " -> ", path.Clean(strings.Replace(ev.Name, *strip, "", -1)))
				} else {
					// moved in from an unwatched location
					log.Println("new:	", path.Clean(strings.Replace(ev.Name, *strip, "", -1)))
				}
				prev.from = false
			case inotify.IN_MOVED_FROM:
				prev.from = true
				prev.name = ev.Name
				prev.when = time.Seconds()
			case inotify.IN_DELETE:
				log.Println("removed: ", path.Clean(strings.Replace(ev.Name, *strip, "", -1)))
			default:
				if *verbose {
					log.Println("event:", ev)
				}
			}
		case err := <-watcher.Error:
			log.Println("error:", err)
		}
	}
}
