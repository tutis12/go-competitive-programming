package main

import (
	"fmt"
	"os"
)

func Recover() {
	err := recover()
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "panic: %v", err)
	os.Exit(13)
}
