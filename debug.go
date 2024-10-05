package main

import (
	"fmt"
	"os"
	"time"
)

func Recover() {
	err := recover()
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "panic: %v", err)
	os.Exit(13)
}

func printSeconds() {
	start := time.Now()
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		defer Recover()
		for range ticker.C {
			fmt.Fprintf(os.Stderr, "%ds passed\n", (time.Since(start)+time.Second/2)/time.Second)
		}
	}()
}
