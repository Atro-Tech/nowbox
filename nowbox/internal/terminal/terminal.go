package terminal

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nowbox/nowbox/internal/adapter"
	"golang.org/x/term"
)

type ProxyOptions struct {
	Modes    []string     // modes available to switch to
	SaveFunc func() error // generates a .now file
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Proxy runs the CLI terminal with a bottom toolbar.
// Returns the next mode to switch to, or "" on quit.
func Proxy(stream adapter.Stream, sessionName, hostAgent string, opts *ProxyOptions) (string, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("setting raw mode: %w", err)
	}
	defer func() {
		fmt.Fprintf(os.Stdout, "\033[r")    // reset scroll region
		fmt.Fprintf(os.Stdout, "\033[2J")   // clear screen
		fmt.Fprintf(os.Stdout, "\033[1;1H") // cursor home
		term.Restore(fd, oldState)
	}()

	var mu sync.Mutex
	w, h := getSize(fd)

	// Set xterm window title
	title := fmt.Sprintf("%s — %s", sessionName, hostAgent)
	setTitle := func() {
		fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
	}
	setTitle()

	// Setup scroll region and draw toolbar
	setupLayout := func() {
		mu.Lock()
		defer mu.Unlock()
		w, h = getSize(fd)
		fmt.Fprintf(os.Stdout, "\033[1;%dr", h-1)
		renderBar(w, h, sessionName, hostAgent, opts)
	}
	setupLayout()
	fmt.Fprintf(os.Stdout, "\033[1;1H")
	stream.Resize(w, h-1)

	// Resize handler
	watchResize(fd, func() {
		setTitle()
		setupLayout()
		mu.Lock()
		cw, ch := w, h
		mu.Unlock()
		stream.Resize(cw, ch-1)
	})

	done := make(chan struct{})
	var once sync.Once
	nextMode := ""
	finish := func() { once.Do(func() { close(done) }) }

	// Remote → stdout
	// Track alt-screen: agent enters alt-screen on start, exits on quit.
	// Only tear down after seeing enter THEN exit.
	inAltScreen := false
	go func() {
		buf := make([]byte, 32*1024)
		count := 0
		for {
			n, readErr := stream.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
				count++
				if count%50 == 0 {
					setTitle()
					mu.Lock()
					cw, ch := w, h
					mu.Unlock()
					renderBar(cw, ch, sessionName, hostAgent, opts)
				}

				chunk := string(buf[:n])
				if contains(chunk, "\033[?1049h") {
					inAltScreen = true
				}
				if inAltScreen && contains(chunk, "\033[?1049l") {
					// Agent exited alt-screen — it quit
					finish()
					return
				}
			}
			if readErr != nil {
				finish()
				return
			}
		}
	}()

	// Stdin → remote
	go func() {
		buf := make([]byte, 1024)
		for {
			n, readErr := os.Stdin.Read(buf)
			if readErr != nil {
				finish()
				return
			}

			out := make([]byte, 0, n)
			for i := 0; i < n; i++ {
				switch buf[i] {
				case 0x11: // Ctrl-Q: quit
					finish()
					return
				case 0x13: // Ctrl-S: save
					if opts != nil && opts.SaveFunc != nil {
						go opts.SaveFunc()
					}
				case 0x1C: // Ctrl-\: switch mode
					if opts != nil && len(opts.Modes) > 0 {
						nextMode = opts.Modes[0]
						finish()
						return
					}
				default:
					out = append(out, buf[i])
				}
			}

			if len(out) > 0 {
				if _, writeErr := stream.Write(out); writeErr != nil {
					finish()
					return
				}
			}
		}
	}()

	<-done
	return nextMode, nil
}

func getSize(fd int) (int, int) {
	w, h, err := term.GetSize(fd)
	if err != nil || h < 3 {
		h = 24
	}
	if err != nil || w < 10 {
		w = 80
	}
	return w, h
}

func renderBar(width, height int, session, hostAgent string, opts *ProxyOptions) {
	fmt.Fprintf(os.Stdout, "\0337")              // save cursor
	fmt.Fprintf(os.Stdout, "\033[%d;1H", height) // move to bottom row

	left := fmt.Sprintf(" %s │ %s", session, hostAgent)

	var shortcuts []string
	if opts != nil && opts.SaveFunc != nil {
		shortcuts = append(shortcuts, "^S Save")
	}
	if opts != nil && len(opts.Modes) > 0 {
		shortcuts = append(shortcuts, "^\\ Switch")
	}
	shortcuts = append(shortcuts, "^Q Quit")
	right := " " + strings.Join(shortcuts, "  ") + " "

	gap := width - runeLen(left) - runeLen(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	if runeLen(bar) > width {
		runes := []rune(bar)
		bar = string(runes[:width])
	}

	fmt.Fprintf(os.Stdout, "\033[7m%s\033[0m", bar) // reverse video
	fmt.Fprintf(os.Stdout, "\0338")                  // restore cursor
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
