//go:build !windows

package terminal

import (
	"os"
	"os/signal"
	"syscall"
)

func watchResize(fd int, onResize func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			onResize()
		}
	}()
}
