package debug

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func Go(fn func()) {
	go func() {
		defer ExitOnPanic()
		fn()
	}()
}

func Try(fn func()) error {
	var err error
	func() {
		defer func() {
			err = RecoverPanic()
		}()
		fn()
	}()
	return err
}

func ExitOnPanic() {
	err := recover()
	if err == nil {
		return
	}
	defer os.Exit(13)

	buf := make([]byte, 10000)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	fmt.Fprintf(os.Stderr, "panic: %v\nstacktrace:\n%s", err, string(buf))
}

func RecoverPanic() error {
	err := recover()
	if err == nil {
		return nil
	}

	buf := make([]byte, 10000)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	return fmt.Errorf("panic: %v\nstacktrace:\n%s", err, string(buf))
}

func PrintSeconds() {
	start := time.Now()
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		defer ExitOnPanic()
		for range ticker.C {
			fmt.Fprintf(os.Stderr, "%ds passed\n", (time.Since(start)+time.Second/2)/time.Second)
		}
	}()
}
