package terminal

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nowbox/nowbox/internal/adapter"
	"golang.org/x/term"
)

// ExitAction tells the caller what to do after the session ends.
type ExitAction int

const (
	ExitDestroy ExitAction = iota // tear down the sandbox
	ExitKeep                      // leave sandbox running (save/update .now file)
)

type ProxyOptions struct {
	Modes        []string
	SaveFunc     func() error
	IsPersistent bool // true when opened from a .now file
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func Proxy(stream adapter.Stream, sessionName string, hostAgent string, opts *ProxyOptions) (string, ExitAction, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", ExitDestroy, fmt.Errorf("setting raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	w, h, _ := term.GetSize(fd)

	title := fmt.Sprintf("%s — %s", sessionName, hostAgent)
	setTitle := func() {
		fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
	}

	setTitle()
	if err := sendResize(stream, fd); err != nil {
		return "", ExitDestroy, err
	}

	// Draw the toolbar on the last row (purely cosmetic — no scroll region, no resize change)
	renderBar(w, h, sessionName, hostAgent, opts)

	watchResize(stream, fd, func() {
		setTitle()
		w, h, _ = term.GetSize(fd)
		renderBar(w, h, sessionName, hostAgent, opts)
	})

	exitAction := ExitDestroy
	if opts != nil && opts.IsPersistent {
		exitAction = ExitKeep
	}

	done := make(chan struct{})
	var mu sync.Mutex
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }
	finishWith := func(action ExitAction) {
		mu.Lock()
		exitAction = action
		mu.Unlock()
		finish()
	}

	// Remote → stdout, detect agent exit
	inAltScreen := false
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
					if !inAltScreen {
						renderBar(w, h, sessionName, hostAgent, opts)
					}
				}

				chunk := string(buf[:n])
				if contains(chunk, "\033[?1049h") {
					inAltScreen = true
				}
				if inAltScreen && contains(chunk, "\033[?1049l") {
					finish()
					return
				}
			}
			if err != nil {
				finish()
				return
			}
		}
	}()

	// Stdin → remote, intercept ctrl+q, ctrl+d, ctrl+s
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				finish()
				return
			}
			for i := 0; i < n; i++ {
				switch buf[i] {
				case 0x11: // ctrl+q — smart quit
					if opts != nil && opts.IsPersistent {
						finishWith(ExitKeep)
					} else {
						finishWith(ExitDestroy)
					}
					return
				case 0x04: // ctrl+d — always destroy
					finishWith(ExitDestroy)
					return
				case 0x13: // ctrl+s — always save/keep
					finishWith(ExitKeep)
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
	mu.Lock()
	action := exitAction
	mu.Unlock()
	return "", action, nil
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

func renderBar(width, height int, session, hostAgent string, opts *ProxyOptions) {
	fmt.Fprintf(os.Stdout, "\0337")              // save cursor
	fmt.Fprintf(os.Stdout, "\033[%d;1H", height) // move to last row

	left := fmt.Sprintf(" ⧉ nowbox | %s | %s", session, hostAgent)

	var shortcuts []string
	if opts != nil && opts.SaveFunc != nil {
		shortcuts = append(shortcuts, "^S Save")
	}
	if opts != nil && opts.IsPersistent {
		shortcuts = append(shortcuts, "^D Destroy")
	}
	if opts != nil && len(opts.Modes) > 0 {
		shortcuts = append(shortcuts, "^\\ Switch")
	}
	shortcuts = append(shortcuts, "^Q Quit")
	right := " " + strings.Join(shortcuts, "  ") + " "

	gap := width - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	if len(bar) > width {
		bar = bar[:width]
	}

	fmt.Fprintf(os.Stdout, "\033[7m%s\033[0m", bar)
	fmt.Fprintf(os.Stdout, "\0338") // restore cursor
}
