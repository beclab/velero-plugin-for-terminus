//go:build !windows
// +build !windows

package log

import (
	"log"
	"os"

	"golang.org/x/sys/unix"
)

func CrashLog(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err.Error())
	} else {
		unix.Dup2(int(f.Fd()), 2)
	}
}
