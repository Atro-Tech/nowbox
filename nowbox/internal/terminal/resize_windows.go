//go:build windows

package terminal

import (
	"time"

	"golang.org/x/term"
)

func watchResize(fd int, onResize func()) {
	go func() {
		lastW, lastH, _ := term.GetSize(fd)
		for {
			time.Sleep(500 * time.Millisecond)
			w, h, err := term.GetSize(fd)
			if err != nil {
				continue
			}
			if w != lastW || h != lastH {
				lastW, lastH = w, h
				onResize()
			}
		}
	}()
}
