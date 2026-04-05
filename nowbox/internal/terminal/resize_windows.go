//go:build windows

package terminal

import (
	"time"

	"github.com/nowbox/nowbox/internal/adapter"
	"golang.org/x/term"
)

func watchResize(stream adapter.Stream, fd int, onResize func()) {
	// Windows has no SIGWINCH. Poll for size changes.
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
				sendResize(stream, fd)
				onResize()
			}
		}
	}()
}
