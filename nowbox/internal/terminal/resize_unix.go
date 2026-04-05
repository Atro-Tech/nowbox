//go:build !windows

package terminal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nowbox/nowbox/internal/adapter"
)

func watchResize(stream adapter.Stream, fd int, onResize func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			sendResize(stream, fd)
			onResize()
		}
	}()
}
