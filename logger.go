package main

import (
	"fmt"
	"os"
	"strings"
)

func Panic(s string, a ...interface{}) {
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	if !strings.HasPrefix(s, "\n") {
		if !strings.HasPrefix(s, "ERROR: ") {
			s = "ERROR: " + s
		}
		s = "\n" + s
	} else {
		if !strings.HasPrefix(s, "\nERROR: ") {
			s = "\nERROR: " + s
		}
	}

	if len(a) == 0 {
		fmt.Printf(s)
	} else {
		fmt.Printf(s, a)
	}
	os.Exit(1)
}
