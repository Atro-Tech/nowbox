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
		fmt.Fprintf(os.Stdout, "\033[?25h") // show cursor
		fmt.Fprintf(os.Stdout, "\033[0m")   // reset attributes
		term.Restore(fd, oldState)
		fmt.Fprintf(os.Stdout, "\r\n")
	}()

	var mu sync.Mutex
	w, h := getSize(fd)
	inAltScreen := false
	pickingMode := false
	pickIndex := 0

	title := fmt.Sprintf("%s — %s", sessionName, hostAgent)
	setTitle := func() {
		fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
	}
	setTitle()

	prevH := h
	showToolbar := func() {
		mu.Lock()
		defer mu.Unlock()
		oldH := prevH
		w, h = getSize(fd)
		if oldH != h && oldH > 0 {
			fmt.Fprintf(os.Stdout, "\0337")
			fmt.Fprintf(os.Stdout, "\033[r")
			fmt.Fprintf(os.Stdout, "\033[%d;1H\033[2K", oldH)
			fmt.Fprintf(os.Stdout, "\0338")
		}
		prevH = h
		fmt.Fprintf(os.Stdout, "\033[1;%dr", h-1)
		renderNormalBar(w, h, sessionName, hostAgent, opts)
	}

	hideToolbar := func() {
		mu.Lock()
		defer mu.Unlock()
		w, h = getSize(fd)
		prevH = h
		fmt.Fprintf(os.Stdout, "\033[r")
	}

	// Start with toolbar visible
	fmt.Fprintf(os.Stdout, "\033[2J\033[1;1H")
	showToolbar()
	fmt.Fprintf(os.Stdout, "\033[1;1H")
	stream.Resize(w, h-1)

	watchResize(fd, func() {
		setTitle()
		mu.Lock()
		alt := inAltScreen
		mu.Unlock()
		if alt {
			mu.Lock()
			w, h = getSize(fd)
			mu.Unlock()
			stream.Resize(w, h)
		} else {
			showToolbar()
			mu.Lock()
			cw, ch := w, h
			mu.Unlock()
			stream.Resize(cw, ch-1)
		}
	})

	done := make(chan struct{})
	var once sync.Once
	nextMode := ""
	finish := func() { once.Do(func() { close(done) }) }

	// Remote → stdout
	go func() {
		buf := make([]byte, 32*1024)
		count := 0
		for {
			n, readErr := stream.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
				count++

				mu.Lock()
				alt := inAltScreen
				picking := pickingMode
				mu.Unlock()

				if !alt && !picking && count%50 == 0 {
					setTitle()
					mu.Lock()
					cw, ch := w, h
					mu.Unlock()
					renderNormalBar(cw, ch, sessionName, hostAgent, opts)
				}

				chunk := string(buf[:n])

				// Detect alt-screen enter — hide toolbar, give full terminal
				if !alt && contains(chunk, "\033[?1049h") {
					mu.Lock()
					inAltScreen = true
					mu.Unlock()
					hideToolbar()
					mu.Lock()
					cw, ch := w, h
					mu.Unlock()
					stream.Resize(cw, ch)
				}

				// Detect alt-screen exit — agent quit
				mu.Lock()
				alt = inAltScreen
				mu.Unlock()
				if alt && contains(chunk, "\033[?1049l") {
					mu.Lock()
					inAltScreen = false
					mu.Unlock()
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

			mu.Lock()
			picking := pickingMode
			mu.Unlock()

			if picking {
				handlePickerInput(buf[:n], &mu, &pickingMode, &pickIndex, &nextMode, opts, w, h, sessionName, hostAgent, finish)
				continue
			}

			out := make([]byte, 0, n)
			for i := 0; i < n; i++ {
				// Filter terminal focus events
				if buf[i] == 0x1B && i+2 < n && buf[i+1] == '[' && (buf[i+2] == 'I' || buf[i+2] == 'O') {
					i += 2
					continue
				}

				switch buf[i] {
				case 0x11: // Ctrl-Q
					finish()
					return
				case 0x13: // Ctrl-S
					if opts != nil && opts.SaveFunc != nil {
						go opts.SaveFunc()
					}
				case 0x1C: // Ctrl-\
					if opts != nil && len(opts.Modes) > 0 {
						mu.Lock()
						pickingMode = true
						pickIndex = 0
						renderPickerBar(w, h, opts.Modes, 0)
						mu.Unlock()
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

func handlePickerInput(input []byte, mu *sync.Mutex, pickingMode *bool, pickIndex *int, nextMode *string, opts *ProxyOptions, w, h int, session, hostAgent string, finish func()) {
	modes := opts.Modes
	for i := 0; i < len(input); i++ {
		b := input[i]

		if b == 0x1B && i+2 < len(input) && input[i+1] == '[' {
			switch input[i+2] {
			case 'A', 'D':
				mu.Lock()
				if *pickIndex > 0 {
					*pickIndex--
				}
				renderPickerBar(w, h, modes, *pickIndex)
				mu.Unlock()
			case 'B', 'C':
				mu.Lock()
				if *pickIndex < len(modes)-1 {
					*pickIndex++
				}
				renderPickerBar(w, h, modes, *pickIndex)
				mu.Unlock()
			}
			i += 2
			continue
		}

		switch b {
		case '\t', 0x1C:
			mu.Lock()
			*pickIndex = (*pickIndex + 1) % len(modes)
			renderPickerBar(w, h, modes, *pickIndex)
			mu.Unlock()
		case '\r', '\n':
			mu.Lock()
			*nextMode = modes[*pickIndex]
			*pickingMode = false
			mu.Unlock()
			finish()
			return
		case 0x1B:
			mu.Lock()
			*pickingMode = false
			renderNormalBar(w, h, session, hostAgent, opts)
			mu.Unlock()
		case 0x11:
			finish()
			return
		}
	}
}

// --- Bar rendering ---

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

func renderNormalBar(width, height int, session, hostAgent string, opts *ProxyOptions) {
	fmt.Fprintf(os.Stdout, "\0337")
	fmt.Fprintf(os.Stdout, "\033[%d;1H", height)

	left := fmt.Sprintf(" ⧉ nowbox | %s | %s", session, hostAgent)

	var shortcuts []string
	if opts != nil && opts.SaveFunc != nil {
		shortcuts = append(shortcuts, "^S Save")
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
	fmt.Fprintf(os.Stdout, "\0338")
}

func renderPickerBar(width, height int, modes []string, selected int) {
	fmt.Fprintf(os.Stdout, "\0337")
	fmt.Fprintf(os.Stdout, "\033[%d;1H", height)

	label := " Switch to: "
	var modeLabels []string
	for i, m := range modes {
		if i == selected {
			modeLabels = append(modeLabels, fmt.Sprintf("\033[1m[%s]\033[22m", m))
		} else {
			modeLabels = append(modeLabels, m)
		}
	}
	left := label + strings.Join(modeLabels, "  ")
	right := " Enter  Esc "

	leftVis := len(label)
	for i, m := range modes {
		if i == selected {
			leftVis += len(m) + 2
		} else {
			leftVis += len(m)
		}
		if i < len(modes)-1 {
			leftVis += 2
		}
	}

	gap := width - leftVis - len(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	fmt.Fprintf(os.Stdout, "\033[7m%s\033[0m", bar)
	fmt.Fprintf(os.Stdout, "\0338")
}
