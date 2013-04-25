package main

import (
	"bufio"
	"fmt"
	"os"
)

const (
	benignError = "Benign error\n"
	nastyError = "Nasty error\n"
)

// simulates an interactive program, like an sqlite session, that 
// reads on stdin, and writes on stdout and sometimes on stderr.
func main() {
	rd := bufio.NewReader(os.Stdin)
	odd := true
	i := 0
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if line != "" {
			if i == 4 || i == 7 {
				fmt.Fprintf(os.Stdout, "\n")
				fmt.Fprintf(os.Stderr, "at %d: %v", i, nastyError)
			} else {
				fmt.Fprintf(os.Stdout, "Nice output: %d\n", i)
				if odd {
					fmt.Fprint(os.Stderr, benignError)
				}
			}
		}
		odd = !odd
		i++
	}
}

