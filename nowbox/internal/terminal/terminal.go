package terminal

import (
	"fmt"
	"os"
	"sync"

	"github.com/nowbox/nowbox/internal/adapter"
	"golang.org/x/term"
)

func Proxy(stream adapter.Stream, sessionName string, hostAgent string) error {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("setting raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	title := fmt.Sprintf("%s — %s", sessionName, hostAgent)
	setTitle := func() {
		fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
	}

	setTitle()
	if err := sendResize(stream, fd); err != nil {
		return err
	}

	watchResize(stream, fd, setTitle)

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	// Remote → stdout
	go func() {
		buf := make([]byte, 32*1024)
		count := 0
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
				count++
				if count%50 == 0 {
					setTitle()
				}
			}
			if err != nil {
				finish()
				return
			}
		}
	}()

	// Stdin → remote, intercept ctrl-q
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				finish()
				return
			}
			for i := 0; i < n; i++ {
				if buf[i] == 0x11 {
					finish()
					return
				}
			}
			if _, err := stream.Write(buf[:n]); err != nil {
				finish()
				return
			}
		}
	}()

	<-done
	return nil
}

func sendResize(stream adapter.Stream, fd int) error {
	w, h, err := term.GetSize(fd)
	if err != nil {
		return nil
	}
	if err := stream.Resize(w, h); err != nil {
		return fmt.Errorf("sending resize: %w", err)
	}
	return nil
}
