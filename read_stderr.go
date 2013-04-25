package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const (
	benignError = "Benign error\n"
	nastyError  = "Nasty error\n"
)

var mu sync.Mutex

type childInfo struct {
	r    *bufio.Reader  // to read the child's stdout
	w    io.WriteCloser // to write to the child's stdin
	proc *os.Process
	c    chan string   // error messages from the child
	pr   *bufio.Reader // to read the child's stdout

}

func startChild() (*childInfo, error) {
	pr1, pw1, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	pr2, pw2, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	pr3, pw3, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmdPath := "/home/mpl/camlistore.org/bar"
	args := []string{cmdPath,}
	fds := []*os.File{pr1, pw2, pw3}
	p, err := os.StartProcess(cmdPath, args, &os.ProcAttr{Dir: "/", Files: fds})
	if err != nil {
		return nil, err
	}
	c := make(chan string)

	go func() {
		brd := bufio.NewReader(pr3)
		for {
			errStr, err := brd.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			if errStr != benignError {
				log.Fatalf("stopped because %v", errStr)
			}
			//println(errStr)
			c <- errStr
		}
	}()

	return &childInfo{
		r:    bufio.NewReader(pr2),
		w:    pw1,
		proc: p,
		c:    c,
	}, nil
}

func main() {
	ci, err := startChild()
	if err != nil {
		panic(err)
	}
	i := 0
	for {
		go func(nb int) {
			select {
			case errStr, ok := <-ci.c:
				if ok {
					if errStr != benignError {
						log.Fatalf("error received by %d from stderr: %v", nb, errStr)
					}
				} else {
					log.Fatal("channel closed")
				}
			default:
			}
			mu.Lock()
			_, err := ci.w.Write([]byte("hello\n"))
			if err != nil {
				log.Fatalf("Error from write: %v", err)
			}
			out, err := ci.r.ReadString('\n')
			if err != nil {
				log.Fatalf("error while reading response: %v", err)
			}
			mu.Unlock()
			select {
			case errStr, ok := <-ci.c:
				if ok {
					if errStr != benignError {
						log.Fatalf("error received by %d from stderr: %v", nb, errStr)
					}
				} else {
					log.Fatal("channel closed")
				}
			default:
			}
			println(out)
		}(i)
		i++
		time.Sleep(100 * time.Microsecond)
	}
}
